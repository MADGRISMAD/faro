package main

// Redes, fase 1: publicar en una página de Facebook cada artículo que Faro publique.
// El cliente autoriza con Facebook Login; se guarda solo el token de la PÁGINA, cifrado (el de usuario no se conserva).
// La cola es la tabla social_posts: la llena un disparador de Postgres al publicar y la vacía socialTick en el worker.
// Variables: FACEBOOK_APP_ID, FACEBOOK_APP_SECRET, TOKEN_ENCRYPTION_KEY y PUBLIC_BASE_URL.
// FACEBOOK_MODE=mock simula Facebook (pruebas locales y E2E); FACEBOOK_GRAPH_URL apunta a otro servidor en las pruebas de Go.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const fbScope = "pages_show_list,pages_read_engagement,pages_manage_posts"

func fbVersion() string { return env("FACEBOOK_GRAPH_VERSION", "v23.0") }
func fbGraph() string {
	return strings.TrimRight(env("FACEBOOK_GRAPH_URL", "https://graph.facebook.com/"+fbVersion()), "/")
}
func fbMock() bool { return env("FACEBOOK_MODE", "") == "mock" }
func fbRedirect() string {
	return env("FACEBOOK_REDIRECT_URI", strings.TrimRight(env("PUBLIC_BASE_URL", ""), "/")+"/api/facebook/callback")
}
func fbEnabled() bool {
	return gKey() != nil && (fbMock() || (env("FACEBOOK_APP_ID", "") != "" && env("FACEBOOK_APP_SECRET", "") != ""))
}

// fbError: lo que Facebook contestó, clasificado para decidir si reintentar, reconectar o rendirse.
type fbError struct {
	Kind string // "auth": token caído o sin permiso · "retry": falla temporal · "unknown": no se sabe si publicó · "fail"
	Msg  string
}

func (e *fbError) Error() string { return e.Msg }

// fbCall habla con la Graph API (o con la simulación). Firma cada llamada con appsecret_proof cuando lleva token.
func fbCall(ctx context.Context, method, path string, v url.Values, out any) error {
	if fbMock() {
		return fbMockCall(path, v, out)
	}
	if tok := v.Get("access_token"); tok != "" {
		m := hmac.New(sha256.New, []byte(env("FACEBOOK_APP_SECRET", "")))
		m.Write([]byte(tok))
		v.Set("appsecret_proof", hex.EncodeToString(m.Sum(nil)))
	}
	endpoint, body := fbGraph()+"/"+path, io.Reader(nil)
	if method == "GET" {
		endpoint += "?" + v.Encode()
	} else {
		body = strings.NewReader(v.Encode())
	}
	req, e := http.NewRequestWithContext(ctx, method, endpoint, body)
	if e != nil {
		return &fbError{"fail", e.Error()}
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	res, e := (&http.Client{Timeout: 25 * time.Second}).Do(req)
	if e != nil { // sin respuesta: en un POST no se sabe si Facebook lo publicó
		return &fbError{"unknown", "Facebook no respondió a tiempo"}
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode == 200 {
		if json.Unmarshal(data, out) != nil {
			return &fbError{"unknown", "Respuesta de Facebook ilegible"}
		}
		return nil
	}
	var wire struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(data, &wire)
	msg := clean(wire.Error.Message, 200)
	if msg == "" {
		msg = fmt.Sprintf("Facebook respondió con estado %d", res.StatusCode)
	}
	switch c := wire.Error.Code; {
	case c == 190 || c == 10 || (c >= 200 && c < 300):
		return &fbError{"auth", msg}
	case c == 1 || c == 2 || c == 4 || c == 17 || c == 32 || c == 341 || res.StatusCode >= 500:
		return &fbError{"retry", msg}
	}
	return &fbError{"fail", msg}
}

// Simulación: dos páginas y publicaciones que siempre salen bien.
func fbMockCall(path string, v url.Values, out any) error {
	var body string
	switch {
	case path == "oauth/access_token":
		body = `{"access_token":"mock-user-token"}`
	case path == "me/accounts":
		body = `{"data":[{"id":"1001","name":"Página de prueba","access_token":"mock-page-1","tasks":["CREATE_CONTENT"]},{"id":"1002","name":"Otra página","access_token":"mock-page-2","tasks":["CREATE_CONTENT"]}]}`
	case strings.HasSuffix(path, "/feed"):
		body = fmt.Sprintf(`{"id":"%s_%d"}`, strings.TrimSuffix(path, "/feed"), time.Now().UnixNano())
	default:
		return &fbError{"fail", "ruta simulada desconocida"}
	}
	return json.Unmarshal([]byte(body), out)
}

type fbPage struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Token string   `json:"access_token"`
	Tasks []string `json:"tasks"`
}

func (a *App) facebookStart(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if !fbEnabled() {
		fail(w, 503, "La conexión con Facebook todavía no está configurada en este servidor")
		return
	}
	var one int
	if e := a.DB.QueryRowContext(r.Context(), `SELECT 1 FROM sites WHERE id=$1 AND tenant_id=$2`, site, t).Scan(&one); e != nil {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	state := signState(site, t)
	if fbMock() { // sin Facebook de por medio: directo al callback
		jsonResponse(w, 200, map[string]any{"url": fbRedirect() + "?" + url.Values{"code": {"mock"}, "state": {state}}.Encode()})
		return
	}
	q := url.Values{"client_id": {env("FACEBOOK_APP_ID", "")}, "redirect_uri": {fbRedirect()}, "state": {state}, "scope": {fbScope}, "response_type": {"code"}}
	jsonResponse(w, 200, map[string]any{"url": "https://www.facebook.com/" + fbVersion() + "/dialog/oauth?" + q.Encode()})
}

func (a *App) facebookCallback(w http.ResponseWriter, r *http.Request) {
	back := strings.TrimRight(env("PUBLIC_BASE_URL", ""), "/") + "/dashboard/redes?facebook="
	site, tenant, ok := readState(r.URL.Query().Get("state"))
	if !ok || !fbEnabled() {
		http.Redirect(w, r, back+"error", 302)
		return
	}
	code := r.URL.Query().Get("code")
	if r.URL.Query().Get("error") != "" || code == "" {
		http.Redirect(w, r, back+"denied", 302)
		return
	}
	var one int
	if e := a.DB.QueryRowContext(r.Context(), `SELECT 1 FROM sites WHERE id=$1 AND tenant_id=$2`, site, tenant).Scan(&one); e != nil {
		http.Redirect(w, r, back+"error", 302)
		return
	}
	app := url.Values{"client_id": {env("FACEBOOK_APP_ID", "")}, "client_secret": {env("FACEBOOK_APP_SECRET", "")}}
	var short, long struct {
		Token string `json:"access_token"`
	}
	v := url.Values{"redirect_uri": {fbRedirect()}, "code": {code}}
	for k, x := range app {
		v[k] = x
	}
	if e := fbCall(r.Context(), "GET", "oauth/access_token", v, &short); e != nil || short.Token == "" {
		slog.Error("Facebook: código", "error", e)
		http.Redirect(w, r, back+"error", 302)
		return
	}
	// token de usuario de larga duración: las páginas que se piden con él traen tokens que no caducan solos
	v = url.Values{"grant_type": {"fb_exchange_token"}, "fb_exchange_token": {short.Token}}
	for k, x := range app {
		v[k] = x
	}
	if e := fbCall(r.Context(), "GET", "oauth/access_token", v, &long); e != nil || long.Token == "" {
		slog.Error("Facebook: token largo", "error", e)
		http.Redirect(w, r, back+"error", 302)
		return
	}
	var list struct {
		Data []fbPage `json:"data"`
	}
	if e := fbCall(r.Context(), "GET", "me/accounts", url.Values{"fields": {"id,name,access_token,tasks"}, "limit": {"100"}, "access_token": {long.Token}}, &list); e != nil {
		slog.Error("Facebook: páginas", "error", e)
		http.Redirect(w, r, back+"error", 302)
		return
	}
	pages := []fbPage{}
	for _, p := range list.Data {
		for _, task := range p.Tasks {
			if task == "CREATE_CONTENT" && p.Token != "" {
				pages = append(pages, p)
				break
			}
		}
	}
	if len(pages) == 0 {
		http.Redirect(w, r, back+"nopages", 302)
		return
	}
	if len(pages) == 1 {
		if e := a.saveFacebookPage(r.Context(), site, tenant, pages[0]); e != nil {
			http.Redirect(w, r, back+"error", 302)
			return
		}
		http.Redirect(w, r, back+"ok", 302)
		return
	}
	blob, _ := json.Marshal(pages)
	enc, e := seal(blob)
	if e == nil {
		_, e = a.DB.ExecContext(r.Context(), `INSERT INTO site_facebook(site_id,tenant_id,status,choices_enc,choices_until) VALUES($1,$2,'choosing',$3,now()+interval '30 minutes')
			ON CONFLICT(site_id) DO UPDATE SET choices_enc=EXCLUDED.choices_enc,choices_until=EXCLUDED.choices_until,status=CASE WHEN site_facebook.status='connected' THEN 'connected' ELSE 'choosing' END`, site, tenant, enc)
	}
	if e != nil {
		http.Redirect(w, r, back+"error", 302)
		return
	}
	http.Redirect(w, r, back+"elegir", 302)
}

func (a *App) saveFacebookPage(ctx context.Context, site, tenant string, p fbPage) error {
	enc, e := seal([]byte(p.Token))
	if e != nil {
		return e
	}
	// cambiar de página cancela lo pendiente para la anterior
	if _, e = a.DB.ExecContext(ctx, `UPDATE social_posts SET status='cancelled',updated_at=now() WHERE site_id=$1 AND status='pending' AND page_id<>$2`, site, p.ID); e != nil {
		return e
	}
	_, e = a.DB.ExecContext(ctx, `INSERT INTO site_facebook(site_id,tenant_id,status,page_id,page_name,token_enc,connected_at) VALUES($1,$2,'connected',$3,$4,$5,now())
		ON CONFLICT(site_id) DO UPDATE SET status='connected',page_id=EXCLUDED.page_id,page_name=EXCLUDED.page_name,token_enc=EXCLUDED.token_enc,choices_enc=NULL,choices_until=NULL,last_error=NULL,connected_at=now()`, site, tenant, p.ID, clean(p.Name, 200), enc)
	return e
}

// choices: las páginas que el usuario puede elegir tras autorizar (vencen a los 30 min).
func (a *App) facebookChoices(ctx context.Context, site, tenant string) []fbPage {
	var enc []byte
	if a.DB.QueryRowContext(ctx, `SELECT choices_enc FROM site_facebook WHERE site_id=$1 AND tenant_id=$2 AND choices_enc IS NOT NULL AND choices_until>now()`, site, tenant).Scan(&enc) != nil {
		return nil
	}
	plain, e := unseal(enc)
	var pages []fbPage
	if e != nil || json.Unmarshal(plain, &pages) != nil {
		return nil
	}
	return pages
}

func (a *App) facebookStatus(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	out := map[string]any{"configured": fbEnabled(), "status": "none"}
	var status, page string
	var lastErr sql.NullString
	e := a.DB.QueryRowContext(r.Context(), `SELECT status,page_name,last_error FROM site_facebook WHERE site_id=$1 AND tenant_id=$2`, site, t).Scan(&status, &page, &lastErr)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		dbError(w, e)
		return
	}
	if e == nil {
		out["status"], out["page_name"], out["error"] = status, page, lastErr.String
	}
	choices := []map[string]string{}
	for _, p := range a.facebookChoices(r.Context(), site, t) {
		choices = append(choices, map[string]string{"id": p.ID, "name": p.Name}) // los tokens nunca salen del servidor
	}
	out["choices"] = choices
	posts := []map[string]any{}
	rows, e := a.DB.QueryContext(r.Context(), `SELECT a.title,p.status,COALESCE(p.post_id,''),COALESCE(p.last_error,''),p.updated_at FROM social_posts p JOIN articles a ON a.id=p.article_id WHERE p.site_id=$1 AND p.tenant_id=$2 ORDER BY p.created_at DESC LIMIT 12`, site, t)
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var title, st, id, msg string
		var at time.Time
		if e = rows.Scan(&title, &st, &id, &msg, &at); e != nil {
			dbError(w, e)
			return
		}
		link := ""
		if id != "" {
			link = "https://www.facebook.com/" + id
		}
		posts = append(posts, map[string]any{"title": title, "status": st, "url": link, "error": msg, "at": at})
	}
	out["posts"] = posts
	// publicados que aún no salieron en Facebook (los de antes de conectar, o los que fallaron o se cancelaron)
	older := []map[string]any{}
	rows2, e := a.DB.QueryContext(r.Context(), `SELECT a.id,a.title,a.published_at FROM articles a LEFT JOIN social_posts p ON p.article_id=a.id
		WHERE a.site_id=$1 AND a.tenant_id=$2 AND a.status='published' AND (p.article_id IS NULL OR p.status IN ('failed','cancelled','unknown')) ORDER BY a.published_at DESC LIMIT 50`, site, t)
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows2.Close()
	for rows2.Next() {
		var id, title string
		var at sql.NullTime
		if e = rows2.Scan(&id, &title, &at); e != nil {
			dbError(w, e)
			return
		}
		older = append(older, map[string]any{"id": id, "title": title, "published_at": at.Time})
	}
	out["unshared"] = older
	jsonResponse(w, 200, out)
}

// facebookShare comparte artículos ya publicados. Varios a la vez salen espaciados (uno cada 2 horas) para no
// llenar la página de golpe; el primero sale en el siguiente turno del worker.
func (a *App) facebookShare(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var x struct {
		IDs []string `json:"article_ids"`
	}
	if !decode(w, r, &x) {
		return
	}
	if len(x.IDs) == 0 || len(x.IDs) > 50 {
		fail(w, 400, "Elige entre 1 y 50 artículos")
		return
	}
	var status string
	if a.DB.QueryRowContext(r.Context(), `SELECT status FROM site_facebook WHERE site_id=$1 AND tenant_id=$2`, site, t).Scan(&status); status != "connected" {
		fail(w, 409, "Primero conecta una página de Facebook")
		return
	}
	queued := 0
	for _, id := range x.IDs {
		id, ok := parseUUID(id)
		if !ok {
			continue
		}
		res, e := a.DB.ExecContext(r.Context(), `INSERT INTO social_posts(article_id,site_id,tenant_id,page_id,updated_at)
			SELECT a.id,a.site_id,a.tenant_id,f.page_id,now()+make_interval(hours => 2*$4) FROM articles a JOIN site_facebook f ON f.site_id=a.site_id AND f.status='connected'
			WHERE a.id=$1 AND a.site_id=$2 AND a.tenant_id=$3 AND a.status='published'
			ON CONFLICT(article_id) DO UPDATE SET status='pending',attempts=0,last_error=NULL,post_id=NULL,page_id=EXCLUDED.page_id,updated_at=EXCLUDED.updated_at
			WHERE social_posts.status IN ('failed','cancelled','unknown')`, id, site, t, queued)
		if e != nil {
			dbError(w, e)
			return
		}
		if n, _ := res.RowsAffected(); n > 0 {
			queued++
		}
	}
	jsonResponse(w, 200, map[string]any{"queued": queued})
}

func (a *App) facebookChoose(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var x struct {
		PageID string `json:"page_id"`
	}
	if !decode(w, r, &x) {
		return
	}
	for _, p := range a.facebookChoices(r.Context(), site, t) {
		if p.ID == x.PageID {
			if e := a.saveFacebookPage(r.Context(), site, t, p); e != nil {
				dbError(w, e)
				return
			}
			jsonResponse(w, 200, map[string]any{"ok": true, "page_name": p.Name})
			return
		}
	}
	fail(w, 409, "Esa página ya no está disponible; vuelve a conectar Facebook")
}

func (a *App) facebookDisconnect(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	// local: revocar el permiso en Facebook podría romper otros sitios que el mismo usuario conectó
	if _, e := a.DB.ExecContext(r.Context(), `DELETE FROM site_facebook WHERE site_id=$1 AND tenant_id=$2`, site, t); e != nil {
		dbError(w, e)
		return
	}
	a.DB.ExecContext(r.Context(), `UPDATE social_posts SET status='cancelled',updated_at=now() WHERE site_id=$1 AND tenant_id=$2 AND status='pending'`, site, t)
	jsonResponse(w, 200, map[string]any{"ok": true})
}

// facebookMessage: el texto del post sale del título y el resumen ya revisados; el enlace va aparte y Facebook arma la tarjeta con la portada.
func facebookMessage(title, summary string) string {
	return clean(strings.TrimSpace(title+"\n\n"+summary), 1800)
}

// socialTick publica lo pendiente. Marca «sending» ANTES de llamar a Facebook: si el worker muere a mitad, queda
// «unknown» en vez de publicarse dos veces.
func (a *App) socialTick(ctx context.Context) error {
	if _, e := a.DB.ExecContext(ctx, `UPDATE social_posts SET status='unknown',last_error='Se interrumpió el envío; revisa tu página antes de reintentar',updated_at=now() WHERE status='sending' AND updated_at<now()-interval '10 minutes'`); e != nil {
		return e
	}
	if !fbEnabled() {
		return nil
	}
	rows, e := a.DB.QueryContext(ctx, `UPDATE social_posts SET status='sending',attempts=attempts+1,updated_at=now() WHERE article_id IN (
		SELECT article_id FROM social_posts WHERE status='pending' AND updated_at<=now() ORDER BY created_at LIMIT 5 FOR UPDATE SKIP LOCKED) RETURNING article_id,attempts`)
	if e != nil {
		return e
	}
	type item struct {
		id       string
		attempts int
	}
	var items []item
	for rows.Next() {
		var it item
		if e = rows.Scan(&it.id, &it.attempts); e != nil {
			rows.Close()
			return e
		}
		items = append(items, it)
	}
	rows.Close()
	for _, it := range items {
		a.sendSocial(ctx, it.id, it.attempts)
	}
	return nil
}

func (a *App) sendSocial(ctx context.Context, article string, attempts int) {
	var title, summary, slug, status, canonical, prefix, domain, pageID, wantPage, fbStatus string
	var connected bool
	var enc []byte
	e := a.DB.QueryRowContext(ctx, `SELECT a.title,a.summary,a.slug,a.status,s.canonical_url,s.blog_prefix,s.domain,s.connected_at IS NOT NULL,p.page_id,COALESCE(f.page_id,''),COALESCE(f.status,''),f.token_enc
		FROM social_posts p JOIN articles a ON a.id=p.article_id JOIN sites s ON s.id=p.site_id LEFT JOIN site_facebook f ON f.site_id=p.site_id WHERE p.article_id=$1`, article).
		Scan(&title, &summary, &slug, &status, &canonical, &prefix, &domain, &connected, &wantPage, &pageID, &fbStatus, &enc)
	set := func(st, msg, post string) {
		if _, err := a.DB.ExecContext(ctx, `UPDATE social_posts SET status=$2,last_error=NULLIF($3,''),post_id=NULLIF($4,''),updated_at=CASE WHEN $2='pending' THEN now()+interval '10 minutes' ELSE now() END WHERE article_id=$1`, article, st, msg, post); err != nil {
			slog.Error("Facebook: guardar resultado", "error", err)
		}
	}
	if e != nil {
		set("failed", "No se encontró el artículo", "")
		return
	}
	// el artículo dejó de estar publicado, o la página se desconectó o cambió: no se publica
	if status != "published" || fbStatus != "connected" || pageID != wantPage {
		set("cancelled", "", "")
		return
	}
	token, e := unseal(enc)
	if e != nil {
		set("failed", "No se pudo leer la conexión con Facebook", "")
		return
	}
	link := blogBase(connected, canonical, prefix, domain) + "/" + slug
	var res struct {
		ID string `json:"id"`
	}
	err := fbCall(ctx, "POST", pageID+"/feed", url.Values{"message": {facebookMessage(title, summary)}, "link": {link}, "access_token": {string(token)}}, &res)
	var fe *fbError
	switch {
	case err == nil:
		set("sent", "", res.ID)
	case errors.As(err, &fe) && fe.Kind == "auth":
		a.DB.ExecContext(ctx, `UPDATE site_facebook SET status='broken',last_error=$2 WHERE site_id=(SELECT site_id FROM social_posts WHERE article_id=$1)`, article, "Facebook ya no deja publicar: vuelve a conectar la página")
		set("failed", fe.Msg, "")
	case errors.As(err, &fe) && fe.Kind == "retry" && attempts < 2:
		set("pending", fe.Msg, "")
	case errors.As(err, &fe) && fe.Kind == "unknown":
		set("unknown", fe.Msg+"; revisa tu página antes de reintentar", "")
	default:
		set("failed", err.Error(), "")
	}
}
