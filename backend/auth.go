package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/argon2"
)

type limitEntry struct {
	Start time.Time
	Count int
}
type Limiter struct {
	mu      sync.Mutex
	entries map[string]limitEntry
}

func NewLimiter() *Limiter { return &Limiter{entries: map[string]limitEntry{}} }
func (l *Limiter) Allow(k string, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.entries) > 10000 {
		for key, v := range l.entries {
			if time.Since(v.Start) > window {
				delete(l.entries, key)
			}
		}
	}
	v := l.entries[k]
	if time.Since(v.Start) > window {
		v = limitEntry{Start: time.Now()}
	}
	v.Count++
	l.entries[k] = v
	return v.Count <= max
}
func token() string        { b := make([]byte, 32); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func passwordHash(p string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	key := argon2.IDKey([]byte(p), salt, 3, 64*1024, 4, 32)
	return "v1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
}
func verifyPassword(p, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 3 || parts[0] != "v1" {
		return false
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[1])
	if e != nil {
		return false
	}
	expected, e := base64.RawStdEncoding.DecodeString(parts[2])
	if e != nil {
		return false
	}
	got := argon2.IDKey([]byte(p), salt, 3, 64*1024, 4, 32)
	return subtle.ConstantTimeCompare(expected, got) == 1
}
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		fail(w, 400, "JSON inválido o demasiado extenso")
		return false
	}
	var extra any
	if e := d.Decode(&extra); !errors.Is(e, io.EOF) {
		fail(w, 400, "JSON inválido")
		return false
	}
	return true
}
func (a *App) originOK(r *http.Request) bool {
	return r.Header.Get("Origin") == env("PUBLIC_BASE_URL", "http://localhost:18080")
}
func clientIP(r *http.Request) string {
	h, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		return r.RemoteAddr
	}
	return h
}
func (a *App) auth(w http.ResponseWriter, r *http.Request, write bool) (string, string, bool) {
	if write && !a.originOK(r) {
		fail(w, 403, "Origen no permitido")
		return "", "", false
	}
	c, e := r.Cookie("faro_session")
	if e != nil {
		fail(w, 401, "Inicia sesión")
		return "", "", false
	}
	var user, tenant, csrfHash string
	e = a.DB.QueryRowContext(r.Context(), `SELECT u.id,u.tenant_id,s.csrf_hash FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id_hash=$1 AND s.expires_at>now()`, hash(c.Value)).Scan(&user, &tenant, &csrfHash)
	if e != nil {
		fail(w, 401, "Sesión vencida")
		return "", "", false
	}
	if write && subtle.ConstantTimeCompare([]byte(hash(r.Header.Get("x-csrf-token"))), []byte(csrfHash)) != 1 {
		fail(w, 403, "Token CSRF inválido")
		return "", "", false
	}
	return user, tenant, true
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	if !a.originOK(r) {
		fail(w, 403, "Origen no permitido")
		return
	}
	if !a.Limiter.Allow("register:"+clientIP(r), 5, time.Hour) {
		fail(w, 429, "Demasiados intentos. Intenta más tarde.")
		return
	}
	var x credentials
	if !decode(w, r, &x) {
		return
	}
	x.Email = strings.ToLower(strings.TrimSpace(x.Email))
	if !valid(x.Email, 254) || !strings.Contains(x.Email, "@") || len(x.Password) < 12 || len(x.Password) > 128 {
		fail(w, 400, "Correo o contraseña inválidos; usa al menos 12 caracteres")
		return
	}
	tx, e := a.DB.BeginTx(r.Context(), nil)
	if e != nil {
		dbError(w, e)
		return
	}
	defer tx.Rollback()
	var tenant, user string
	e = tx.QueryRowContext(r.Context(), `INSERT INTO tenants(name) VALUES($1) RETURNING id`, x.Email).Scan(&tenant)
	if e != nil {
		dbError(w, e)
		return
	}
	e = tx.QueryRowContext(r.Context(), `INSERT INTO users(tenant_id,email,password_hash) VALUES($1,$2,$3) RETURNING id`, tenant, x.Email, passwordHash(x.Password)).Scan(&user)
	if e != nil {
		var pg *pgconn.PgError
		if errors.As(e, &pg) && pg.Code == "23505" {
			fail(w, 409, "El correo ya está registrado")
		} else {
			dbError(w, e)
		}
		return
	}
	if e = tx.Commit(); e != nil {
		dbError(w, e)
		return
	}
	a.session(w, r, user, tenant)
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if !a.originOK(r) {
		fail(w, 403, "Origen no permitido")
		return
	}
	if !a.Limiter.Allow("login:"+clientIP(r), 10, 15*time.Minute) {
		fail(w, 429, "Demasiados intentos. Intenta más tarde.")
		return
	}
	var x credentials
	if !decode(w, r, &x) {
		return
	}
	var user, tenant, pass string
	e := a.DB.QueryRowContext(r.Context(), `SELECT id,tenant_id,password_hash FROM users WHERE email=$1`, strings.ToLower(strings.TrimSpace(x.Email))).Scan(&user, &tenant, &pass)
	if e != nil || !verifyPassword(x.Password, pass) {
		fail(w, 401, "Credenciales incorrectas")
		return
	}
	a.session(w, r, user, tenant)
}
func (a *App) session(w http.ResponseWriter, r *http.Request, user, tenant string) {
	session, csrf := token(), token()
	_, e := a.DB.ExecContext(r.Context(), `INSERT INTO sessions(id_hash,user_id,csrf_hash,expires_at) VALUES($1,$2,$3,now()+interval '30 days')`, hash(session), user, hash(csrf))
	if e != nil {
		dbError(w, e)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "faro_session", Value: session, Path: "/", MaxAge: 2592000, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: strings.HasPrefix(env("PUBLIC_BASE_URL", ""), "https://")})
	jsonResponse(w, 200, map[string]any{"user_id": user, "tenant_id": tenant, "csrf": csrf})
}
func (a *App) me(w http.ResponseWriter, r *http.Request) {
	user, tenant, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	var budget, spent, reserved int64
	e := a.DB.QueryRowContext(r.Context(), `SELECT budget_microusd,spent_microusd,reserved_microusd FROM tenants WHERE id=$1`, tenant).Scan(&budget, &spent, &reserved)
	if e != nil {
		dbError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"user_id": user, "tenant_id": tenant, "budget_microusd": budget, "spent_microusd": spent, "reserved_microusd": reserved})
}
func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	_, _, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	if c, e := r.Cookie("faro_session"); e == nil {
		_, _ = a.DB.ExecContext(r.Context(), `DELETE FROM sessions WHERE id_hash=$1`, hash(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "faro_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	jsonResponse(w, 200, map[string]bool{"ok": true})
}
func parseUUID(s string) (string, bool) {
	id, e := uuid.Parse(s)
	if e != nil {
		return "", false
	}
	return id.String(), true
}
func requireUUID(w http.ResponseWriter, s string) (string, bool) {
	id, ok := parseUUID(s)
	if !ok {
		fail(w, 400, "UUID inválido")
	}
	return id, ok
}
