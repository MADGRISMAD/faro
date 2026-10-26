package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Config struct {
	Model, Key            string
	InputRate, OutputRate int64
	MaxOutput             int
	ImageModel            string // "" = sin imágenes
	ImageRate             int64  // USD por millón de tokens de salida de imagen, ×1000
}
type App struct {
	DB       *sql.DB
	Provider Provider
	Config   Config
	Limiter  *Limiter
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func parseRate(s string) (int64, error) {
	parts := strings.Split(s, ".")
	if len(parts) > 2 || len(parts[0]) == 0 {
		return 0, fmt.Errorf("tarifa inválida")
	}
	whole, e := strconv.ParseInt(parts[0], 10, 64)
	if e != nil || whole < 0 {
		return 0, fmt.Errorf("tarifa inválida")
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > 3 {
		return 0, fmt.Errorf("máximo tres decimales")
	}
	for len(frac) < 3 {
		frac += "0"
	}
	f, e := strconv.ParseInt(frac, 10, 64)
	if e != nil {
		return 0, e
	}
	return whole*1000 + f, nil
}
func config() (Config, error) {
	in, e := parseRate(os.Getenv("GEMINI_INPUT_USD_PER_MILLION"))
	if e != nil {
		return Config{}, e
	}
	out, e := parseRate(os.Getenv("GEMINI_OUTPUT_USD_PER_MILLION"))
	if e != nil {
		return Config{}, e
	}
	max, e := strconv.Atoi(env("GEMINI_MAX_OUTPUT_TOKENS", "3000"))
	if e != nil || max < 100 || max > 8000 {
		return Config{}, errors.New("GEMINI_MAX_OUTPUT_TOKENS inválido")
	}
	model := env("GEMINI_MODEL", "mock")
	img := env("GEMINI_IMAGE_MODEL", "gemini-3.1-flash-lite-image") // el más barato de Nano Banana
	if model == "mock" {
		img = "mock"
	} else if img == "off" {
		img = ""
	}
	imgRate, e := parseRate(env("GEMINI_IMAGE_USD_PER_MILLION", "30.0"))
	if e != nil {
		return Config{}, e
	}
	return Config{model, os.Getenv("GEMINI_API_KEY"), in, out, max, img, imgRate}, nil
}
func migrate(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(20261020)`); e != nil {
		return e
	}
	var exists sql.NullString
	if e = tx.QueryRowContext(ctx, `SELECT to_regclass('public.tenants')`).Scan(&exists); e != nil {
		return e
	}
	if !exists.Valid {
		b, readErr := migrations.ReadFile("migrations/20261020000000_init.sql")
		if readErr != nil {
			return readErr
		}
		if _, e = tx.ExecContext(ctx, string(b)); e != nil {
			return e
		}
	}
	b, readErr := migrations.ReadFile("migrations/20261020000001_worker_lease.sql")
	if readErr != nil {
		return readErr
	}
	if _, e = tx.ExecContext(ctx, string(b)); e != nil {
		return e
	}
	b, readErr = migrations.ReadFile("migrations/20261025000000_research.sql")
	if readErr != nil {
		return readErr
	}
	if _, e = tx.ExecContext(ctx, string(b)); e != nil {
		return e
	}
	b, readErr = migrations.ReadFile("migrations/20261026000000_images.sql")
	if readErr != nil {
		return readErr
	}
	if _, e = tx.ExecContext(ctx, string(b)); e != nil {
		return e
	}
	b, readErr = migrations.ReadFile("migrations/20261027000000_connect.sql")
	if readErr != nil {
		return readErr
	}
	if _, e = tx.ExecContext(ctx, string(b)); e != nil {
		return e
	}
	b, readErr = migrations.ReadFile("migrations/20261028000000_wizard.sql")
	if readErr != nil {
		return readErr
	}
	if _, e = tx.ExecContext(ctx, string(b)); e != nil {
		return e
	}
	b, readErr = migrations.ReadFile("migrations/20261029000000_data.sql")
	if readErr != nil {
		return readErr
	}
	if _, e = tx.ExecContext(ctx, string(b)); e != nil {
		return e
	}
	b, readErr = migrations.ReadFile("migrations/20261030000000_facebook.sql")
	if readErr != nil {
		return readErr
	}
	if _, e = tx.ExecContext(ctx, string(b)); e != nil {
		return e
	}
	b, readErr = migrations.ReadFile("migrations/20261031000000_slug_alias.sql")
	if readErr != nil {
		return readErr
	}
	if _, e = tx.ExecContext(ctx, string(b)); e != nil {
		return e
	}
	return tx.Commit()
}
func main() {
	cfg, e := config()
	if e != nil {
		slog.Error("Configuración inválida", "error", e)
		os.Exit(1)
	}
	db, e := sql.Open("pgx", env("DATABASE_URL", "postgres://faro:faro@localhost:15432/faro"))
	if e != nil {
		slog.Error("Postgres", "error", e)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(10 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if e = db.PingContext(ctx); e != nil {
		slog.Error("Postgres no disponible", "error", e)
		os.Exit(1)
	}
	if e = migrate(db); e != nil {
		slog.Error("Migración falló", "error", e)
		os.Exit(1)
	}
	p := Provider(&Mock{})
	if cfg.Model != "mock" && cfg.Key != "" {
		p = &Gemini{Client: &http.Client{Timeout: 85 * time.Second}}
		slog.Info("Gemini activo", "model", cfg.Model)
	} else {
		slog.Info("Proveedor simulado activo")
	}
	app := &App{db, p, cfg, NewLimiter()}
	if env("MODE", "server") == "worker" {
		app.Work(context.Background())
		return
	}
	addr := env("BIND_ADDR", "0.0.0.0:18081")
	slog.Info("Servidor listo", "addr", addr)
	if e = http.ListenAndServe(addr, app.Routes()); e != nil {
		slog.Error("Servidor detenido", "error", e)
		os.Exit(1)
	}
}
func fail(w http.ResponseWriter, code int, msg string) {
	jsonResponse(w, code, map[string]any{"error": msg})
}
func dbError(w http.ResponseWriter, e error) {
	slog.Error("Error interno", "error", e)
	fail(w, 500, "Error interno")
}
func pathUUID(s string) (string, error) {
	if len(s) != 36 {
		return "", fmt.Errorf("UUID inválido")
	}
	return s, nil
}
func valid(s string, max int) bool {
	return strings.TrimSpace(s) != "" && len(s) <= max && !strings.ContainsRune(s, 0)
}
