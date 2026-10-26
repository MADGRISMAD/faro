package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	ok := map[string]string{"www.Ejemplo.com": "https://www.ejemplo.com", "ejemplo.com/blog?x=1#a": "https://ejemplo.com", "http://ejemplo.com.mx/": "http://ejemplo.com.mx", " https://EJEMPLO.com:443/ ": "https://ejemplo.com", "//ejemplo.com": "https://ejemplo.com"}
	for in, want := range ok {
		u, e := normalizeURL(in)
		if e != nil || u.String() != want {
			t.Errorf("%q -> %v %v, quería %s", in, u, e, want)
		}
	}
	for _, in := range []string{"", "localhost", "127.0.0.1", "http://10.0.0.1/", "ftp://ejemplo.com", "ejemplo.com:8080", "https://user:pw@ejemplo.com", "ej emplo.com", "[::1]"} {
		if u, e := normalizeURL(in); e == nil {
			t.Errorf("%q debía rechazarse, dio %v", in, u)
		}
	}
}

func TestPublicIP(t *testing.T) {
	for ip, want := range map[string]bool{"8.8.8.8": true, "2606:4700::1111": true, "127.0.0.1": false, "10.1.2.3": false, "192.168.1.1": false, "172.16.0.1": false, "169.254.169.254": false, "100.64.0.1": false, "0.0.0.0": false, "::1": false, "fe80::1": false, "fc00::1": false} {
		if publicIP(net.ParseIP(ip)) != want {
			t.Errorf("%s: esperaba %v", ip, want)
		}
	}
}

func TestSSRFBloqueaDireccionesPrivadas(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("<html>secreto</html>")) }))
	defer s.Close()
	if _, _, _, e := readSite(context.Background(), s.URL); e == nil {
		t.Fatal("leyó un servidor en 127.0.0.1")
	}
	// una redirección hacia la red interna también debe fallar
	r := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "http://169.254.169.254/latest", 302)
	}))
	defer r.Close()
	if _, _, _, e := readSite(context.Background(), r.URL); e == nil {
		t.Fatal("siguió una redirección a metadata")
	}
}

func TestInvestigacionAutomaticaYAutopiloto(t *testing.T) {
	allowPrivate, minAutoWords = true, 0
	defer func() { allowPrivate, minAutoWords = false, 500 }()
	var site *httptest.Server
	site = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			fmt.Fprint(w, `<html><head><title>Taller Rápido | Mecánica</title><meta name="description" content="Afinación y frenos en Tijuana"></head><body><h1>Taller Rápido</h1><script>x</script><a href="/servicios">Servicios</a><a href="/logo.png">logo</a><a href="https://otro.com/x">otro</a></body></html>`)
		case "/servicios":
			fmt.Fprint(w, `<html><body><h2>Frenos</h2><p>Cambio de balatas</p></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()
	text, final, platform, e := readSite(context.Background(), site.URL)
	if e != nil || !strings.Contains(text, "Taller Rápido") || !strings.Contains(text, "Cambio de balatas") || strings.Contains(text, "x</script") || strings.Contains(text, "otro.com") {
		t.Fatalf("lectura: %v\n%s", e, text)
	}
	if platform != "other" {
		t.Fatalf("plataforma: %s", platform)
	}
	if final.Host != strings.TrimPrefix(site.URL, "http://") {
		t.Fatalf("url final %v", final)
	}

	app := testApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()
	cookie, csrf, tenant := registerTest(t, srv)
	otherCookie, _, _ := registerTest(t, srv)
	if code, _, _ := request(t, srv, "POST", "/sites/analyze", map[string]string{"url": "tusitio"}, cookie, csrf, ""); code != 400 {
		t.Fatalf("url inválida: %d", code)
	}
	if code, _, _ := request(t, srv, "POST", "/sites/analyze", map[string]string{"url": "ejemplo-" + token()[:6] + ".com"}, cookie, "", ""); code != 403 {
		t.Fatalf("sin CSRF: %d", code)
	}
	host := "taller-" + token()[:8] + ".test"
	code, res, _ := request(t, srv, "POST", "/sites/analyze", map[string]string{"url": "www." + host + "/servicios"}, cookie, csrf, "")
	if code != 200 {
		t.Fatalf("analyze %d %#v", code, res)
	}
	siteID, jobID := res["site_id"].(string), res["job_id"].(string)
	if code, _, _ := request(t, srv, "POST", "/sites/analyze", map[string]string{"url": host}, otherCookie, "", ""); code == 200 {
		t.Fatal("otro cliente registró el mismo dominio sin CSRF")
	}
	if code, _, _ := request(t, srv, "GET", "/sites/"+siteID+"/research", nil, otherCookie, "", ""); code != 404 {
		t.Fatalf("investigación ajena visible: %d", code)
	}
	// el sitio real vive en un servidor local: se apunta el job a él
	if _, e := app.DB.Exec(`UPDATE jobs SET topic=$1 WHERE id=$2`, site.URL, jobID); e != nil {
		t.Fatal(e)
	}
	var j *WorkItem
	for i := 0; i < 20; i++ { // otras pruebas pueden dejar jobs en cola
		if j, e = app.claim(context.Background()); e != nil || j == nil || j.ID == jobID {
			break
		}
	}
	if e != nil || j == nil || j.ID != jobID || j.Kind != "research" {
		t.Fatalf("claim: %v %#v", e, j)
	}
	app.runResearch(context.Background(), j)
	code, st, _ := request(t, srv, "GET", "/sites/"+siteID+"/research", nil, cookie, "", "")
	if code != 200 || st["status"] != "done" || st["research"] == nil || st["autopilot"] != true {
		t.Fatalf("estado: %d %#v", code, st)
	}
	var name, tone string
	var pending int
	app.DB.QueryRow(`SELECT name,tone FROM sites WHERE id=$1`, siteID).Scan(&name, &tone)
	app.DB.QueryRow(`SELECT count(*) FROM jobs WHERE site_id=$1 AND kind='article' AND status='queued'`, siteID).Scan(&pending)
	if name != "Taller Rápido" || tone == "" || pending != 1 {
		t.Fatalf("sitio: %q %q, %d artículos en cola", name, tone, pending)
	}
	var spent int64
	app.DB.QueryRow(`SELECT spent_microusd FROM tenants WHERE id=$1`, tenant).Scan(&spent)
	if spent <= 0 || spent > 50000 {
		t.Fatalf("costo registrado %d", spent)
	}
	// el primer artículo se escribe y, con autopiloto, sale publicado sin pedir nada
	for i := 0; i < 20; i++ {
		if j, e = app.claim(context.Background()); e != nil || j == nil || j.Site == siteID {
			break
		}
	}
	if e != nil || j == nil || j.Kind != "article" {
		t.Fatalf("claim artículo: %v %#v", e, j)
	}
	if e = app.finish(context.Background(), j, app.Provider.Create(context.Background(), j.Prompt, app.Config)); e != nil {
		t.Fatal(e)
	}
	var published int
	app.DB.QueryRow(`SELECT count(*) FROM articles WHERE site_id=$1 AND status='published' AND published_at IS NOT NULL`, siteID).Scan(&published)
	if published != 1 {
		t.Fatalf("autopiloto no publicó: %d", published)
	}
	// pasada una semana, el autopiloto retoma el ritmo con el siguiente tema sin usar
	app.DB.Exec(`UPDATE jobs SET created_at=now()-interval '8 days',status=CASE WHEN status='queued' THEN 'failed' ELSE status END WHERE site_id=$1 AND kind='article'`, siteID)
	if e := app.autopilotTick(context.Background()); e != nil {
		t.Fatal(e)
	}
	var again int
	app.DB.QueryRow(`SELECT count(*) FROM jobs WHERE site_id=$1 AND kind='article' AND status='queued'`, siteID).Scan(&again)
	if again != 1 {
		t.Fatalf("el autopiloto no retomó el ritmo: %d", again)
	}
	app.DB.Exec(`UPDATE jobs SET status='failed' WHERE site_id=$1 AND status='queued'`, siteID)
	// apagar el autopiloto vuelve al flujo con aprobación
	if code, _, _ := request(t, srv, "PATCH", "/sites/"+siteID+"/autopilot", map[string]any{"enabled": false, "posts_per_week": 2}, cookie, csrf, ""); code != 200 {
		t.Fatalf("autopilot off: %d", code)
	}
	// el ritmo semanal no encola de más: ya hay 3 artículos esta semana
	if e := app.autopilotTick(context.Background()); e != nil {
		t.Fatal(e)
	}
	var total int
	app.DB.QueryRow(`SELECT count(*) FROM jobs WHERE site_id=$1 AND kind='article'`, siteID).Scan(&total)
	if total != 2 {
		t.Fatalf("encoló de más: %d", total)
	}
}

func TestAutopilotNoPublicaSensibleNiFlaco(t *testing.T) {
	app := testApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()
	cookie, csrf, tenant := registerTest(t, srv)
	_, res, _ := request(t, srv, "POST", "/sites/analyze", map[string]string{"url": "clinica-" + token()[:8] + ".test"}, cookie, csrf, "")
	siteID := res["site_id"].(string)
	app.DB.Exec(`UPDATE jobs SET status='failed' WHERE site_id=$1 AND kind='research'`, siteID) // la investigación no es lo que se prueba aquí
	run := func(_ int) string {                                                                 // escribe un artículo del sitio y devuelve su estado
		var id string
		app.DB.QueryRow(`INSERT INTO jobs(tenant_id,site_id,idempotency_key,topic,business,audience,kind) VALUES($1,$2,$3,'Tema','Negocio','Público','article') RETURNING id`, tenant, siteID, token()).Scan(&id)
		var j *WorkItem
		for i := 0; i < 30; i++ {
			j, _ = app.claim(context.Background())
			if j == nil || j.ID == id {
				break
			}
		}
		if j == nil || j.ID != id {
			t.Fatal("no se pudo tomar el job")
		}
		if e := app.finish(context.Background(), j, app.Provider.Create(context.Background(), j.Prompt, app.Config)); e != nil {
			t.Fatal(e)
		}
		var st string
		app.DB.QueryRow(`SELECT status FROM articles WHERE job_id=$1`, id).Scan(&st)
		return st
	}
	minAutoWords = 5000 // el artículo simulado es corto: debe quedar en borrador
	if st := run(0); st != "draft" {
		t.Fatalf("artículo corto: %s", st)
	}
	minAutoWords = 0
	if st := run(0); st != "published" {
		t.Fatalf("artículo normal: %s", st)
	}
	app.DB.Exec(`UPDATE sites SET auto_publish=false WHERE id=$1`, siteID) // sector sensible
	if st := run(0); st != "draft" {
		t.Fatalf("sector sensible: %s", st)
	}
	minAutoWords = 500
}

func TestDetectarPlataforma(t *testing.T) {
	h := func(k, v string) http.Header { x := http.Header{}; x.Set(k, v); return x }
	cases := []struct {
		html string
		hdr  http.Header
		want string
	}{
		{`<link href="/wp-content/themes/x.css">`, nil, "wordpress"},
		{`<img src="https://static.wixstatic.com/a.png">`, nil, "wix"},
		{`<script src="https://cdn.shopify.com/s/x.js">`, nil, "shopify"},
		{`<html>`, h("X-Vercel-Id", "iad1::abc"), "vercel"},
		{`<html>`, h("Server", "Netlify"), "netlify"},
		{`<html>`, h("Server", "cloudflare"), "cloudflare"},
		{`<html>`, h("Server", "nginx/1.25"), "nginx"},
		{`<html>`, nil, "other"},
	}
	for _, c := range cases {
		if got := detectPlatform(c.html, c.hdr); got != c.want {
			t.Errorf("%q %v -> %s, quería %s", c.html, c.hdr, got, c.want)
		}
	}
}

func TestVerificarConexionDelBlog(t *testing.T) {
	allowPrivate = true
	defer func() { allowPrivate = false }()
	const id = "11111111-2222-3333-4444-555555555555"
	mux := http.NewServeMux()
	mux.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<html><head><meta name="faro-site" content="%s"></head></html>`, id)
	})
	mux.HandleFunc("/otro", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `<html>otra cosa</html>`) })
	s := httptest.NewServer(mux)
	defer s.Close()
	if ok, why := verifyBlog(context.Background(), s.URL, "/blog", id); !ok {
		t.Fatalf("conectado debía ser verdadero: %s", why)
	}
	if ok, why := verifyBlog(context.Background(), s.URL, "/blog", "99999999-2222-3333-4444-555555555555"); ok || why == "" {
		t.Fatal("aceptó el blog de otro sitio")
	}
	if ok, why := verifyBlog(context.Background(), s.URL, "/otro", id); ok || !strings.Contains(why, "no es el blog") {
		t.Fatalf("página sin marca: %v %s", ok, why)
	}
	if ok, why := verifyBlog(context.Background(), s.URL, "/nada", id); ok || !strings.Contains(why, "404") {
		t.Fatalf("404: %v %s", ok, why)
	}
	if got := blogBase(false, "https://x.com", "/blog", "x.com"); !strings.HasSuffix(got, "/p/x.com") {
		t.Fatalf("sin conectar debe apuntar al blog alojado: %s", got)
	}
	if got := blogBase(true, "https://x.com/", "/blog", "x.com"); got != "https://x.com/blog" {
		t.Fatalf("conectado: %s", got)
	}
}

func TestAutoVerificaLaConexionSola(t *testing.T) {
	allowPrivate = true
	defer func() { allowPrivate = false }()
	app := testApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()
	cookie, csrf, _ := registerTest(t, srv)
	_, res, _ := request(t, srv, "POST", "/sites/analyze", map[string]string{"url": "auto-" + token()[:8] + ".test"}, cookie, csrf, "")
	siteID := res["site_id"].(string)
	mux := http.NewServeMux()
	connected := false
	mux.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		if !connected {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `<meta name="faro-site" content="%s">`, siteID)
	})
	client := httptest.NewServer(mux)
	defer client.Close()
	app.DB.Exec(`UPDATE sites SET canonical_url=$1,research='{}'::jsonb WHERE id=$2`, client.URL, siteID)
	app.DB.Exec(`UPDATE sites SET research=NULL WHERE id<>$1`, siteID) // la base de pruebas acumula sitios de otras corridas; autoVerify revisa solo unos cuantos
	app.autoVerify(context.Background())
	var at *string
	app.DB.QueryRow(`SELECT connected_at::text FROM sites WHERE id=$1`, siteID).Scan(&at)
	if at != nil {
		t.Fatal("marcó conectado un sitio que responde 404")
	}
	connected = true // el cliente publica su cambio
	app.autoVerify(context.Background())
	app.DB.QueryRow(`SELECT connected_at::text FROM sites WHERE id=$1`, siteID).Scan(&at)
	if at == nil {
		t.Fatal("no detectó la conexión sola")
	}
}
