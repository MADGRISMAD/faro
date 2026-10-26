package main

// Conexión del blog con el sitio del cliente: dónde vive el blog, cómo se verifica que quedó conectado
// y a qué enlaza el widget mientras no lo esté (nunca a una página que dé 404).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// hostedBase es la dirección del blog alojado en Faro, que funciona desde el primer minuto sin tocar el sitio del cliente.
func hostedBase(domain string) string {
	return strings.TrimRight(env("PUBLIC_BASE_URL", ""), "/") + "/p/" + domain
}

// blogBase: el blog en el dominio del cliente si ya se conectó; si no, el alojado en Faro.
func blogBase(connected bool, canonical, prefix, domain string) string {
	if connected {
		return strings.TrimRight(canonical, "/") + prefix
	}
	return hostedBase(domain)
}

// verifyBlog abre canonical+prefix y busca la marca <meta name="faro-site"> del sitio. Devuelve (conectado, motivo).
func verifyBlog(ctx context.Context, canonical, prefix, siteID string) (bool, string) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	target := strings.TrimRight(canonical, "/") + prefix
	req, e := http.NewRequestWithContext(ctx, "GET", target, nil)
	if e != nil {
		return false, "La dirección del blog no es válida."
	}
	req.Header.Set("User-Agent", "FaroAiBot/1.0 (verificación de conexión)")
	res, e := safeClient().Do(req)
	if e != nil {
		return false, "No pude abrir " + target + ". Revisa que tu sitio esté en línea."
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return false, fmt.Sprintf("%s responde 404: tu sitio todavía no está enviando esa ruta a Faro Ai.", target)
	}
	if res.StatusCode != 200 {
		return false, fmt.Sprintf("%s respondió con estado %d.", target, res.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 400000))
	if !strings.Contains(string(body), `name="faro-site" content="`+siteID+`"`) {
		return false, target + " abre, pero no es el blog de Faro Ai de este sitio."
	}
	return true, ""
}

// autoVerify revisa solo los sitios que aún no están conectados: en cuanto el cliente publica su cambio, Faro lo detecta
// sin que tenga que volver a apretar «Verificar». (ponytail: sondeo cada tick; si hay miles de sitios, repartir por lotes)
func (a *App) autoVerify(ctx context.Context) error {
	rows, e := a.DB.QueryContext(ctx, `SELECT id,canonical_url,blog_prefix FROM sites WHERE connected_at IS NULL AND research IS NOT NULL AND created_at>now()-interval '60 days' ORDER BY random() LIMIT 8`)
	if e != nil {
		return e
	}
	type pending struct{ id, canonical, prefix string }
	var list []pending
	for rows.Next() {
		var p pending
		if e = rows.Scan(&p.id, &p.canonical, &p.prefix); e != nil {
			rows.Close()
			return e
		}
		list = append(list, p)
	}
	rows.Close()
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second) // un sitio caído no debe atrasar al resto
	defer cancel()
	for _, p := range list {
		if ctx.Err() != nil {
			break
		}
		if ok, _ := verifyBlog(ctx, p.canonical, p.prefix, p.id); ok {
			if _, e = a.DB.ExecContext(ctx, `UPDATE sites SET connected_at=now() WHERE id=$1 AND connected_at IS NULL`, p.id); e != nil {
				return e
			}
		}
	}
	return nil
}

func (a *App) verifySite(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	if !a.Limiter.Allow("verify:"+t, 20, time.Hour) {
		fail(w, 429, "Demasiadas verificaciones. Intenta en un rato.")
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var canonical, prefix string
	e := a.DB.QueryRowContext(r.Context(), `SELECT canonical_url,blog_prefix FROM sites WHERE id=$1 AND tenant_id=$2`, site, t).Scan(&canonical, &prefix)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	okConn, why := verifyBlog(r.Context(), canonical, prefix, site)
	if okConn {
		_, e = a.DB.ExecContext(r.Context(), `UPDATE sites SET connected_at=COALESCE(connected_at,now()) WHERE id=$1`, site)
	} else {
		_, e = a.DB.ExecContext(r.Context(), `UPDATE sites SET connected_at=NULL WHERE id=$1`, site)
	}
	if e != nil {
		dbError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"connected": okConn, "reason": why})
}

// deleteSite borra un sitio con todo lo suyo (artículos, imágenes, trabajos). Pide escribir el dominio para evitar accidentes
// y se niega si hay algo escribiéndose, para no dejar reservas de presupuesto colgadas.
func (a *App) deleteSite(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	tx, e := a.DB.BeginTx(r.Context(), nil)
	if e != nil {
		dbError(w, e)
		return
	}
	defer tx.Rollback()
	var domain string
	e = tx.QueryRowContext(r.Context(), `SELECT domain FROM sites WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, site, t).Scan(&domain)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	if strings.TrimPrefix(strings.ToLower(strings.TrimSpace(r.URL.Query().Get("confirm"))), "www.") != domain {
		fail(w, 400, "Escribe el dominio del sitio para confirmar que quieres quitarlo")
		return
	}
	var running int
	if e = tx.QueryRowContext(r.Context(), `SELECT count(*) FROM jobs WHERE site_id=$1 AND status='running'`, site).Scan(&running); e != nil {
		dbError(w, e)
		return
	}
	if running > 0 {
		fail(w, 409, "La IA está escribiendo algo para este sitio. Intenta de nuevo en un minuto.")
		return
	}
	// Los cobros ya hechos quedan en el consumo del cliente (tenants.spent); aquí solo se borra el detalle por trabajo.
	for _, q := range []string{
		`UPDATE jobs SET article_id=NULL WHERE site_id=$1`,
		`DELETE FROM articles WHERE site_id=$1`,
		`DELETE FROM usage_events WHERE job_id IN (SELECT id FROM jobs WHERE site_id=$1)`,
		`DELETE FROM jobs WHERE site_id=$1`,
		`DELETE FROM sites WHERE id=$1`,
	} {
		if _, e = tx.ExecContext(r.Context(), q, site); e != nil {
			dbError(w, e)
			return
		}
	}
	if e = tx.Commit(); e != nil {
		dbError(w, e)
		return
	}
	jsonResponse(w, 200, map[string]any{"ok": true})
}
