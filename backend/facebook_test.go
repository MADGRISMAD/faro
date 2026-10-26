package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

// Conectar Facebook, elegir página y que cada artículo publicado salga una sola vez; si Facebook revoca el permiso,
// la conexión queda rota y no se reintenta a ciegas.
func TestFacebookConectaYPublica(t *testing.T) {
	var revoked atomic.Bool
	var posts atomic.Int32
	graph := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/access_token":
			w.Write([]byte(`{"access_token":"tok-` + r.Form.Get("grant_type") + `"}`))
		case "/me/accounts":
			if r.Form.Get("access_token") != "tok-fb_exchange_token" || r.Form.Get("appsecret_proof") == "" {
				t.Errorf("las páginas se piden con el token largo y firmado: %v", r.Form)
			}
			w.Write([]byte(`{"data":[{"id":"1001","name":"Mi página","access_token":"page-1","tasks":["CREATE_CONTENT"]},{"id":"1002","name":"Solo lectura","access_token":"page-2","tasks":["ANALYZE"]},{"id":"1003","name":"Otra","access_token":"page-3","tasks":["CREATE_CONTENT","MODERATE"]}]}`))
		case "/1001/feed":
			if revoked.Load() {
				w.WriteHeader(400)
				w.Write([]byte(`{"error":{"message":"Error validating access token","code":190}}`))
				return
			}
			if r.Form.Get("access_token") != "page-1" || !strings.Contains(r.Form.Get("link"), "/blog/") || !strings.HasPrefix(r.Form.Get("message"), "Artículo ") {
				t.Errorf("post: %v", r.Form)
			}
			posts.Add(1)
			w.Write([]byte(`{"id":"1001_555"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer graph.Close()
	t.Setenv("TOKEN_ENCRYPTION_KEY", testKey)
	t.Setenv("FACEBOOK_APP_ID", "app")
	t.Setenv("FACEBOOK_APP_SECRET", "secreto")
	t.Setenv("FACEBOOK_GRAPH_URL", graph.URL)
	app := testApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()
	cookie, csrf, tenant, siteID, _ := siteWithTopics(t, app, srv)
	app.DB.Exec(`UPDATE sites SET connected_at=now(),canonical_url='https://ejemplo.test',blog_prefix='/blog' WHERE id=$1`, siteID)

	publish := func(slug string) {
		var id string
		if e := app.DB.QueryRow(`INSERT INTO articles(tenant_id,site_id,title,slug,meta_description,summary,content,status) VALUES($1,$2,$3,$4,'m','Resumen','{"intro":"x","sections":[],"faq":[]}','draft') RETURNING id`, tenant, siteID, "Artículo "+slug, slug).Scan(&id); e != nil {
			t.Fatal(e)
		}
		app.DB.Exec(`UPDATE articles SET status='approved' WHERE id=$1`, id) // aprobar no comparte
		app.DB.Exec(`UPDATE articles SET status='published',published_at=now() WHERE id=$1`, id)
	}
	old := "fb-viejo-" + token()[:6]
	publish(old) // antes de conectar: no se comparte solo

	_, res, _ := request(t, srv, "POST", "/sites/"+siteID+"/facebook/start", map[string]any{}, cookie, csrf, "")
	u, _ := url.Parse(res["url"].(string))
	state := u.Query().Get("state")
	if !strings.Contains(u.Query().Get("scope"), "pages_manage_posts") || state == "" {
		t.Fatalf("url de autorización: %v", u)
	}
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, e := noFollow.Get(srv.URL + "/facebook/callback?code=abc&state=" + url.QueryEscape(state))
	if e != nil || !strings.HasSuffix(resp.Header.Get("Location"), "facebook=elegir") {
		t.Fatalf("callback: %v %v", e, resp.Header.Get("Location"))
	}
	if resp, _ := noFollow.Get(srv.URL + "/facebook/callback?code=abc&state=falso"); !strings.HasSuffix(resp.Header.Get("Location"), "facebook=error") {
		t.Fatal("aceptó un state falso")
	}
	_, st, _ := request(t, srv, "GET", "/sites/"+siteID+"/facebook", nil, cookie, "", "")
	if ch := st["choices"].([]any); len(ch) != 2 || st["status"] != "choosing" {
		t.Fatalf("debía ofrecer solo las 2 páginas donde puede publicar: %v", st)
	}
	if code, _, _ := request(t, srv, "POST", "/sites/"+siteID+"/facebook/page", map[string]string{"page_id": "1002"}, cookie, csrf, ""); code != 409 {
		t.Fatalf("eligió una página sin permiso de publicar: %d", code)
	}
	if code, _, _ := request(t, srv, "POST", "/sites/"+siteID+"/facebook/page", map[string]string{"page_id": "1001"}, cookie, csrf, ""); code != 200 {
		t.Fatalf("elegir página: %d", code)
	}

	publish("fb-uno-" + token()[:6])
	ctx := context.Background()
	app.socialTick(ctx)
	app.socialTick(ctx) // nada se manda dos veces
	if posts.Load() != 1 {
		t.Fatalf("publicaciones en Facebook: %d", posts.Load())
	}
	// los anteriores se comparten con el botón; uno solo sale de inmediato
	_, st, _ = request(t, srv, "GET", "/sites/"+siteID+"/facebook", nil, cookie, "", "")
	un := st["unshared"].([]any)
	if len(un) != 1 || un[0].(map[string]any)["title"] != "Artículo "+old {
		t.Fatalf("pendientes de compartir: %v", un)
	}
	if code, r, _ := request(t, srv, "POST", "/sites/"+siteID+"/facebook/share", map[string]any{"article_ids": []string{un[0].(map[string]any)["id"].(string)}}, cookie, csrf, ""); code != 200 || r["queued"] != float64(1) {
		t.Fatalf("compartir viejo: %d %v", code, r)
	}
	app.socialTick(ctx)
	if posts.Load() != 2 {
		t.Fatalf("el artículo viejo no salió: %d", posts.Load())
	}
	_, st, _ = request(t, srv, "GET", "/sites/"+siteID+"/facebook", nil, cookie, "", "")
	first := st["posts"].([]any)[0].(map[string]any)
	if first["status"] != "sent" || first["url"] != "https://www.facebook.com/1001_555" || st["page_name"] != "Mi página" {
		t.Fatalf("estado: %v", st)
	}

	revoked.Store(true)
	publish("fb-dos-" + token()[:6])
	app.socialTick(ctx)
	_, st, _ = request(t, srv, "GET", "/sites/"+siteID+"/facebook", nil, cookie, "", "")
	if st["status"] != "broken" || st["posts"].([]any)[0].(map[string]any)["status"] != "failed" {
		t.Fatalf("con el permiso revocado: %v", st)
	}

	// desconectar: lo nuevo ya no entra a la cola
	request(t, srv, "DELETE", "/sites/"+siteID+"/facebook", nil, cookie, csrf, "")
	publish("fb-tres-" + token()[:6])
	var n int
	app.DB.QueryRow(`SELECT count(*) FROM social_posts WHERE site_id=$1`, siteID).Scan(&n)
	if n != 3 {
		t.Fatalf("tras desconectar no debía encolar: %d", n)
	}
}
