package main

// Analytics REAL del sitio con Google Search Console (impresiones, clics, CTR, posición, consultas y páginas).
// El cliente autoriza con su cuenta de Google (solo lectura); el refresh token se guarda cifrado con AES-GCM.
// Variables: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, TOKEN_ENCRYPTION_KEY (64 hex) y PUBLIC_BASE_URL.

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const gScope = "https://www.googleapis.com/auth/webmasters.readonly"

func gAuthURL() string  { return env("GOOGLE_AUTH_URL", "https://accounts.google.com/o/oauth2/v2/auth") }
func gTokenURL() string { return env("GOOGLE_TOKEN_URL", "https://oauth2.googleapis.com/token") }
func gRevokeURL() string {
	return env("GOOGLE_REVOKE_URL", "https://oauth2.googleapis.com/revoke")
}
func gAPI() string {
	return strings.TrimRight(env("GOOGLE_API_BASE", "https://www.googleapis.com/webmasters/v3"), "/")
}
func gRedirect() string {
	return env("GOOGLE_REDIRECT_URI", strings.TrimRight(env("PUBLIC_BASE_URL", ""), "/")+"/api/google/callback")
}
func gKey() []byte { // 32 bytes en hex; sin ella la integración queda apagada
	k, e := hex.DecodeString(env("TOKEN_ENCRYPTION_KEY", ""))
	if e != nil || len(k) != 32 {
		return nil
	}
	return k
}
func gEnabled() bool {
	return env("GOOGLE_CLIENT_ID", "") != "" && env("GOOGLE_CLIENT_SECRET", "") != "" && gKey() != nil
}

func seal(plain []byte) ([]byte, error) {
	b, e := aes.NewCipher(gKey())
	if e != nil {
		return nil, e
	}
	g, e := cipher.NewGCM(b)
	if e != nil {
		return nil, e
	}
	nonce := make([]byte, g.NonceSize())
	if _, e = io.ReadFull(rand.Reader, nonce); e != nil {
		return nil, e
	}
	return g.Seal(nonce, nonce, plain, nil), nil
}

func unseal(data []byte) ([]byte, error) {
	b, e := aes.NewCipher(gKey())
	if e != nil {
		return nil, e
	}
	g, e := cipher.NewGCM(b)
	if e != nil || len(data) < g.NonceSize() {
		return nil, errors.New("dato cifrado inválido")
	}
	return g.Open(nil, data[:g.NonceSize()], data[g.NonceSize():], nil)
}

// El state firma sitio, cliente y caducidad: el callback no necesita sesión y nadie puede conectar el sitio de otro.
func signState(site, tenant string) string {
	p, _ := json.Marshal(map[string]any{"s": site, "t": tenant, "e": time.Now().Add(15 * time.Minute).Unix()})
	m := hmac.New(sha256.New, gKey())
	m.Write(p)
	return base64.RawURLEncoding.EncodeToString(p) + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func readState(s string) (site, tenant string, ok bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 2 || gKey() == nil {
		return "", "", false
	}
	p, e1 := base64.RawURLEncoding.DecodeString(parts[0])
	sig, e2 := base64.RawURLEncoding.DecodeString(parts[1])
	m := hmac.New(sha256.New, gKey())
	m.Write(p)
	if e1 != nil || e2 != nil || !hmac.Equal(sig, m.Sum(nil)) {
		return "", "", false
	}
	var raw map[string]any
	if json.Unmarshal(p, &raw) != nil {
		return "", "", false
	}
	site, _ = raw["s"].(string)
	tenant, _ = raw["t"].(string)
	exp, _ := raw["e"].(float64)
	if time.Now().Unix() > int64(exp) {
		return "", "", false
	}
	return site, tenant, site != "" && tenant != ""
}

func gForm(ctx context.Context, endpoint string, v url.Values, out any) error {
	req, e := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(v.Encode()))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, e := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if e != nil {
		return errors.New("no se pudo contactar a Google")
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != 200 {
		return fmt.Errorf("Google respondió con estado %d", res.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (a *App) googleStart(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if !gEnabled() {
		fail(w, 503, "La conexión con Google todavía no está configurada en este servidor")
		return
	}
	var one int
	if e := a.DB.QueryRowContext(r.Context(), `SELECT 1 FROM sites WHERE id=$1 AND tenant_id=$2`, site, t).Scan(&one); e != nil {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	q := url.Values{"client_id": {env("GOOGLE_CLIENT_ID", "")}, "redirect_uri": {gRedirect()}, "response_type": {"code"}, "scope": {gScope}, "access_type": {"offline"}, "prompt": {"consent"}, "state": {signState(site, t)}}
	jsonResponse(w, 200, map[string]any{"url": gAuthURL() + "?" + q.Encode()})
}

// matchProperty elige la propiedad de Search Console que corresponde al dominio del sitio.
func matchProperty(entries []struct {
	SiteURL         string `json:"siteUrl"`
	PermissionLevel string `json:"permissionLevel"`
}, domain string) string {
	want := []string{"sc-domain:" + domain, "https://" + domain + "/", "https://www." + domain + "/", "http://" + domain + "/", "http://www." + domain + "/"}
	for _, w := range want {
		for _, e := range entries {
			if strings.EqualFold(e.SiteURL, w) && e.PermissionLevel != "siteUnverifiedUser" {
				return e.SiteURL
			}
		}
	}
	return ""
}

func (a *App) googleCallback(w http.ResponseWriter, r *http.Request) {
	back := strings.TrimRight(env("PUBLIC_BASE_URL", ""), "/") + "/dashboard/analytics?google="
	site, tenant, ok := readState(r.URL.Query().Get("state"))
	if !ok || !gEnabled() {
		http.Redirect(w, r, back+"error", 302)
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		http.Redirect(w, r, back+"denied", 302)
		return
	}
	var domain string
	if e := a.DB.QueryRowContext(r.Context(), `SELECT domain FROM sites WHERE id=$1 AND tenant_id=$2`, site, tenant).Scan(&domain); e != nil {
		http.Redirect(w, r, back+"error", 302)
		return
	}
	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if e := gForm(r.Context(), gTokenURL(), url.Values{"code": {r.URL.Query().Get("code")}, "client_id": {env("GOOGLE_CLIENT_ID", "")}, "client_secret": {env("GOOGLE_CLIENT_SECRET", "")}, "redirect_uri": {gRedirect()}, "grant_type": {"authorization_code"}}, &tok); e != nil || tok.RefreshToken == "" {
		http.Redirect(w, r, back+"error", 302)
		return
	}
	var list struct {
		SiteEntry []struct {
			SiteURL         string `json:"siteUrl"`
			PermissionLevel string `json:"permissionLevel"`
		} `json:"siteEntry"`
	}
	if e := gGet(r.Context(), tok.AccessToken, gAPI()+"/sites", &list); e != nil {
		http.Redirect(w, r, back+"error", 302)
		return
	}
	prop := matchProperty(list.SiteEntry, domain)
	enc, e := seal([]byte(tok.RefreshToken))
	if e != nil {
		http.Redirect(w, r, back+"error", 302)
		return
	}
	if _, e = a.DB.ExecContext(r.Context(), `INSERT INTO site_google(site_id,tenant_id,refresh_enc,property) VALUES($1,$2,$3,$4) ON CONFLICT(site_id) DO UPDATE SET refresh_enc=EXCLUDED.refresh_enc,property=EXCLUDED.property,connected_at=now()`, site, tenant, enc, prop); e != nil {
		http.Redirect(w, r, back+"error", 302)
		return
	}
	a.DB.ExecContext(r.Context(), `DELETE FROM analytics_cache WHERE site_id=$1`, site)
	if prop == "" { // la cuenta no tiene este dominio verificado en Search Console
		http.Redirect(w, r, back+"noproperty", 302)
		return
	}
	http.Redirect(w, r, back+"ok", 302)
}

func gGet(ctx context.Context, access, u string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+access)
	return gDo(req, out)
}

func gPost(ctx context.Context, access, u string, body any, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Content-Type", "application/json")
	return gDo(req, out)
}

func gDo(req *http.Request, out any) error {
	res, e := (&http.Client{Timeout: 25 * time.Second}).Do(req)
	if e != nil {
		return errors.New("no se pudo contactar a Google")
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode != 200 {
		return fmt.Errorf("Google respondió con estado %d", res.StatusCode)
	}
	return json.Unmarshal(data, out)
}

type gRow struct {
	Keys        []string `json:"keys"`
	Clicks      float64  `json:"clicks"`
	Impressions float64  `json:"impressions"`
	CTR         float64  `json:"ctr"`
	Position    float64  `json:"position"`
}

func (a *App) googleAccess(ctx context.Context, site string) (access, property string, err error) {
	var enc []byte
	if e := a.DB.QueryRowContext(ctx, `SELECT refresh_enc,property FROM site_google WHERE site_id=$1`, site).Scan(&enc, &property); e != nil {
		return "", "", e
	}
	refresh, e := unseal(enc)
	if e != nil {
		return "", "", e
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if e = gForm(ctx, gTokenURL(), url.Values{"client_id": {env("GOOGLE_CLIENT_ID", "")}, "client_secret": {env("GOOGLE_CLIENT_SECRET", "")}, "refresh_token": {string(refresh)}, "grant_type": {"refresh_token"}}, &tok); e != nil || tok.AccessToken == "" {
		return "", "", errors.New("Google rechazó la conexión: vuelve a conectar tu cuenta")
	}
	return tok.AccessToken, property, nil
}

func (a *App) siteAnalytics(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	days := map[string]int{"7d": 7, "28d": 28, "90d": 90}[r.URL.Query().Get("range")]
	rng := r.URL.Query().Get("range")
	if days == 0 {
		days, rng = 28, "28d"
	}
	var one int
	if e := a.DB.QueryRowContext(r.Context(), `SELECT 1 FROM sites WHERE id=$1 AND tenant_id=$2`, site, t).Scan(&one); e != nil {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	var property string
	var connectedAt time.Time
	e := a.DB.QueryRowContext(r.Context(), `SELECT property,connected_at FROM site_google WHERE site_id=$1`, site).Scan(&property, &connectedAt)
	if errors.Is(e, sql.ErrNoRows) {
		jsonResponse(w, 200, map[string]any{"configured": gEnabled(), "connected": false})
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	if property == "" {
		jsonResponse(w, 200, map[string]any{"configured": gEnabled(), "connected": true, "property": "", "error": "Tu cuenta de Google no tiene este sitio verificado en Search Console. Verifícalo en search.google.com/search-console y vuelve a conectar."})
		return
	}
	var cached []byte
	var at time.Time
	if e := a.DB.QueryRowContext(r.Context(), `SELECT body,fetched_at FROM analytics_cache WHERE site_id=$1 AND range=$2 AND fetched_at>now()-interval '1 hour'`, site, rng).Scan(&cached, &at); e == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}
	access, _, e := a.googleAccess(r.Context(), site)
	if e != nil {
		fail(w, 502, e.Error())
		return
	}
	// Los datos de Search Console llegan con 2-3 días de retraso: el periodo termina hace 3 días.
	end := time.Now().UTC().AddDate(0, 0, -3)
	start := end.AddDate(0, 0, -(days - 1))
	pend := start.AddDate(0, 0, -1)
	pstart := pend.AddDate(0, 0, -(days - 1))
	f := func(d time.Time) string { return d.Format("2006-01-02") }
	endpoint := gAPI() + "/sites/" + strings.ReplaceAll(url.QueryEscape(property), "+", "%20") + "/searchAnalytics/query"
	query := func(from, to string, dims []string, limit int) ([]gRow, error) {
		body := map[string]any{"startDate": from, "endDate": to, "rowLimit": limit}
		if len(dims) > 0 {
			body["dimensions"] = dims
		}
		var out struct {
			Rows []gRow `json:"rows"`
		}
		return out.Rows, gPost(r.Context(), access, endpoint, body, &out)
	}
	total := func(rows []gRow) map[string]any {
		if len(rows) == 0 {
			return map[string]any{"clicks": 0, "impressions": 0, "ctr": 0, "position": 0}
		}
		return map[string]any{"clicks": rows[0].Clicks, "impressions": rows[0].Impressions, "ctr": rows[0].CTR, "position": rows[0].Position}
	}
	cur, e1 := query(f(start), f(end), nil, 1)
	prev, e2 := query(f(pstart), f(pend), nil, 1)
	series, e3 := query(f(start), f(end), []string{"date"}, 120)
	queries, e4 := query(f(start), f(end), []string{"query"}, 25)
	pages, e5 := query(f(start), f(end), []string{"page"}, 25)
	if err := errors.Join(e1, e2, e3, e4, e5); err != nil {
		fail(w, 502, "Google no devolvió los datos: "+err.Error())
		return
	}
	list := func(rows []gRow, key string) []map[string]any {
		out := []map[string]any{}
		for _, x := range rows {
			if len(x.Keys) > 0 {
				out = append(out, map[string]any{key: x.Keys[0], "clicks": x.Clicks, "impressions": x.Impressions, "ctr": x.CTR, "position": x.Position})
			}
		}
		return out
	}
	days2 := []map[string]any{}
	for _, x := range series {
		if len(x.Keys) > 0 {
			days2 = append(days2, map[string]any{"date": x.Keys[0], "clicks": x.Clicks, "impressions": x.Impressions})
		}
	}
	body, _ := json.Marshal(map[string]any{"configured": true, "connected": true, "property": property, "range": rng, "start": f(start), "end": f(end), "totals": total(cur), "previous": total(prev), "series": days2, "queries": list(queries, "query"), "pages": list(pages, "page"), "fetched_at": time.Now()})
	a.DB.ExecContext(r.Context(), `INSERT INTO analytics_cache(site_id,range,body) VALUES($1,$2,$3) ON CONFLICT(site_id,range) DO UPDATE SET body=EXCLUDED.body,fetched_at=now()`, site, rng, body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

func (a *App) googleDisconnect(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var enc []byte
	e := a.DB.QueryRowContext(r.Context(), `SELECT g.refresh_enc FROM site_google g JOIN sites s ON s.id=g.site_id WHERE g.site_id=$1 AND s.tenant_id=$2`, site, t).Scan(&enc)
	if errors.Is(e, sql.ErrNoRows) {
		jsonResponse(w, 200, map[string]any{"ok": true})
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	if gEnabled() { // se revoca el permiso en Google (mejor esfuerzo) antes de olvidar el token
		if tok, e := unseal(enc); e == nil {
			_ = gForm(r.Context(), gRevokeURL(), url.Values{"token": {string(tok)}}, nil)
		}
	}
	a.DB.ExecContext(r.Context(), `DELETE FROM site_google WHERE site_id=$1`, site)
	a.DB.ExecContext(r.Context(), `DELETE FROM analytics_cache WHERE site_id=$1`, site)
	jsonResponse(w, 200, map[string]any{"ok": true})
}
