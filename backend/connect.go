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
	req.Header.Set("User-Agent", "FaroContenidoBot/1.0 (verificación de conexión)")
	res, e := safeClient().Do(req)
	if e != nil {
		return false, "No pude abrir " + target + ". Revisa que tu sitio esté en línea."
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return false, fmt.Sprintf("%s responde 404: tu sitio todavía no está enviando esa ruta a FaroContenido.", target)
	}
	if res.StatusCode != 200 {
		return false, fmt.Sprintf("%s respondió con estado %d.", target, res.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 400000))
	if !strings.Contains(string(body), `name="faro-site" content="`+siteID+`"`) {
		return false, target + " abre, pero no es el blog de FaroContenido de este sitio."
	}
	return true, ""
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
