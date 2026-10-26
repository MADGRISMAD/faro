package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func rawGet(t *testing.T, srv *httptest.Server, path, cookie string) (int, string, int) {
	t.Helper()
	req, _ := http.NewRequest("GET", srv.URL+path, nil)
	req.Header.Set("Origin", "http://localhost:18080")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	buf := make([]byte, 1<<20)
	n, _ := res.Body.Read(buf)
	return res.StatusCode, res.Header.Get("Content-Type"), n
}

func TestImagenDePortada(t *testing.T) {
	minAutoWords = 0
	defer func() { minAutoWords = 500 }()
	app := testApp(t)
	app.Config.ImageModel, app.Config.ImageRate = "mock", 30000
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()
	cookie, csrf, tenant := registerTest(t, srv)
	otherCookie, _, _ := registerTest(t, srv)
	_, res, _ := request(t, srv, "POST", "/sites/analyze", map[string]string{"url": "imagenes-" + token()[:8] + ".test"}, cookie, csrf, "")
	siteID := res["site_id"].(string)
	app.DB.Exec(`UPDATE jobs SET status='failed' WHERE site_id=$1`, siteID)
	app.DB.Exec(`UPDATE sites SET autopilot=true WHERE id=$1`, siteID)
	next := func(want string) *WorkItem { // toma el siguiente job de ESTE cliente
		for i := 0; i < 40; i++ {
			j, e := app.claim(context.Background())
			if e != nil || j == nil {
				break
			}
			if j.Tenant == tenant && j.Kind == want {
				return j
			}
		}
		t.Fatalf("no apareció el job %s", want)
		return nil
	}
	write := func() string { // escribe un artículo y devuelve su id
		var id string
		app.DB.QueryRow(`INSERT INTO jobs(tenant_id,site_id,idempotency_key,topic,business,audience,kind) VALUES($1,$2,$3,'Tema','Negocio','Público','article') RETURNING id`, tenant, siteID, token()).Scan(&id)
		j := next("article")
		if e := app.finish(context.Background(), j, app.Provider.Create(context.Background(), j.Prompt, app.Config)); e != nil {
			t.Fatal(e)
		}
		var art string
		app.DB.QueryRow(`SELECT id FROM articles WHERE job_id=$1`, id).Scan(&art)
		return art
	}
	art := write()
	// el artículo ya está publicado (autopiloto) pero su imagen aún no existe
	if code, _, _ := rawGet(t, srv, "/articles/"+art+"/image", cookie); code != 404 {
		t.Fatalf("imagen inexistente: %d", code)
	}
	img := next("image")
	if img.Article != art || img.Reserve != imageReserve {
		t.Fatalf("job de imagen: %#v", img)
	}
	app.runImage(context.Background(), img)
	var has bool
	var cost int64
	app.DB.QueryRow(`SELECT has_image FROM articles WHERE id=$1`, art).Scan(&has)
	app.DB.QueryRow(`SELECT cost_microusd FROM usage_events WHERE job_id=$1`, img.ID).Scan(&cost)
	if !has || cost != imageCost(app.Config, 100, 1120) || cost > imageReserve {
		t.Fatalf("imagen: has=%v costo=%d", has, cost)
	}
	if code, ct, n := rawGet(t, srv, "/articles/"+art+"/image", cookie); code != 200 || ct != "image/jpeg" || n < 100 {
		t.Fatalf("dueño: %d %s %d", code, ct, n)
	}
	if code, _, _ := rawGet(t, srv, "/articles/"+art+"/image", otherCookie); code != 404 {
		t.Fatalf("imagen ajena visible: %d", code)
	}
	var slug string
	app.DB.QueryRow(`SELECT slug FROM articles WHERE id=$1`, art).Scan(&slug)
	if code, ct, _ := rawGet(t, srv, "/public/sites/"+siteID+"/articles/"+slug+"/image", ""); code != 200 || ct != "image/jpeg" {
		t.Fatalf("pública: %d %s", code, ct)
	}
	app.DB.Exec(`UPDATE articles SET status='draft' WHERE id=$1`, art)
	if code, _, _ := rawGet(t, srv, "/public/sites/"+siteID+"/articles/"+slug+"/image", ""); code != 404 {
		t.Fatalf("la imagen de un borrador no debe ser pública: %d", code)
	}
	// apagada en Brand DNA: no se encola ninguna imagen
	if code, _, _ := request(t, srv, "PATCH", "/sites/"+siteID+"/brand", map[string]any{"images_enabled": false, "image_style": "watercolor", "avoid_topics": "precios"}, cookie, csrf, ""); code != 200 {
		t.Fatalf("brand: %d", code)
	}
	if code, _, _ := request(t, srv, "PATCH", "/sites/"+siteID+"/brand", map[string]any{"image_style": "inventado"}, cookie, csrf, ""); code != 400 {
		t.Fatalf("estilo inválido aceptado: %d", code)
	}
	art2 := write()
	var n int
	app.DB.QueryRow(`SELECT count(*) FROM jobs WHERE article_id=$1 AND kind='image'`, art2).Scan(&n)
	if n != 0 {
		t.Fatalf("encoló imagen con las imágenes apagadas: %d", n)
	}
	// el prompt usa el estilo y no pide texto
	if p := imageText(ImagePrompt{Title: "X", Business: "Y", Audience: "Z", Style: "watercolor", Notes: "incluye un gato"}); !strings.Contains(p, "acuarela") || !strings.Contains(p, "gato") || !strings.Contains(p, "Prohibido absolutamente: texto") {
		t.Fatalf("prompt: %s", p)
	}
	// una imagen corrupta no se guarda y el job se reintenta
	if _, e := toJPEG([]byte("no soy una imagen")); e == nil {
		t.Fatal("aceptó basura como imagen")
	}
	app.DB.Exec(`UPDATE jobs SET status='failed' WHERE tenant_id=$1 AND status IN ('queued','running')`, tenant)
}
