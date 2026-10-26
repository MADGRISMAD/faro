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
	if _, _, e := readSite(context.Background(), s.URL); e == nil {
		t.Fatal("leyó un servidor en 127.0.0.1")
	}
	// una redirección hacia la red interna también debe fallar
	r := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "http://169.254.169.254/latest", 302)
	}))
	defer r.Close()
	if _, _, e := readSite(context.Background(), r.URL); e == nil {
		t.Fatal("siguió una redirección a metadata")
	}
}

func TestInvestigacionAutomaticaYAutopiloto(t *testing.T) {
	allowPrivate = true
	defer func() { allowPrivate = false }()
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
	text, final, e := readSite(context.Background(), site.URL)
	if e != nil || !strings.Contains(text, "Taller Rápido") || !strings.Contains(text, "Cambio de balatas") || strings.Contains(text, "x</script") || strings.Contains(text, "otro.com") {
		t.Fatalf("lectura: %v\n%s", e, text)
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
	if name != "Taller Rápido" || tone == "" || pending != 3 {
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
	if total != 4 {
		t.Fatalf("encoló de más: %d", total)
	}
}
