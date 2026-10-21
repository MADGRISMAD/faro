package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func testApp(t *testing.T) *App {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no configurado")
	}
	db, e := sql.Open("pgx", url)
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(20)
	if e = migrate(db); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	return &App{DB: db, Provider: &Mock{}, Config: Config{Model: "mock", InputRate: 750, OutputRate: 3750, MaxOutput: 3000}, Limiter: NewLimiter()}
}
func request(t *testing.T, server *httptest.Server, method, path string, body any, cookie, csrf, key string) (int, map[string]any, *http.Cookie) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			t.Fatal(e)
		}
		reader = bytes.NewReader(b)
	}
	req, e := http.NewRequest(method, server.URL+path, reader)
	if e != nil {
		t.Fatal(e)
	}
	req.Header.Set("Origin", "http://localhost:18080")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if csrf != "" {
		req.Header.Set("x-csrf-token", csrf)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	res, e := http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	var data map[string]any
	_ = json.NewDecoder(res.Body).Decode(&data)
	var c *http.Cookie
	for _, candidate := range res.Cookies() {
		if candidate.Name == "faro_session" {
			c = candidate
		}
	}
	return res.StatusCode, data, c
}
func registerTest(t *testing.T, s *httptest.Server) (string, string, string) {
	t.Helper()
	email := "test-" + token() + "@example.test"
	code, data, c := request(t, s, "POST", "/auth/register", map[string]string{"email": email, "password": "ClaveSegura2026!"}, "", "", "")
	if code != 200 || c == nil {
		t.Fatalf("registro: %d %#v", code, data)
	}
	return c.Name + "=" + c.Value, data["csrf"].(string), data["tenant_id"].(string)
}
func TestFlujoMockAislamientoIdempotenciaYBorrador(t *testing.T) {
	app := testApp(t)
	server := httptest.NewServer(app.Routes())
	defer server.Close()
	cookie, csrf, tenant := registerTest(t, server)
	otherCookie, _, _ := registerTest(t, server)
	badCode, _, _ := request(t, server, "POST", "/sites", map[string]any{"name": "Sin CSRF"}, cookie, "", "")
	if badCode != 403 {
		t.Fatalf("escritura sin CSRF: %d", badCode)
	}
	code, site, _ := request(t, server, "POST", "/sites", map[string]any{"name": "Taller", "domain": "example-" + token() + ".test", "canonical_url": "http://localhost:18082", "blog_prefix": "/blog", "brand_description": "Prueba", "tone": "Claro"}, cookie, csrf, "")
	if code != 200 {
		t.Fatalf("sitio %d %#v", code, site)
	}
	siteID := site["id"].(string)
	payload := map[string]any{"topic": "Mantenimiento", "business": "Taller", "audience": "Conductores", "keyword": nil}
	_, eBudget := app.DB.Exec(`UPDATE tenants SET budget_microusd=0 WHERE id=$1`, tenant)
	if eBudget != nil {
		t.Fatal(eBudget)
	}
	budgetCode, budgetResponse, _ := request(t, server, "POST", "/sites/"+siteID+"/generations", payload, cookie, csrf, "budget")
	if budgetCode != 402 || budgetResponse["error"] != "Presupuesto agotado" {
		t.Fatalf("presupuesto: %d %#v", budgetCode, budgetResponse)
	}
	_, eBudget = app.DB.Exec(`UPDATE tenants SET budget_microusd=500000 WHERE id=$1`, tenant)
	if eBudget != nil {
		t.Fatal(eBudget)
	}
	code, j, _ := request(t, server, "POST", "/sites/"+siteID+"/generations", payload, cookie, csrf, "key-1")
	if code != 200 {
		t.Fatalf("generación: %d %#v", code, j)
	}
	id := j["job_id"].(string)
	code, same, _ := request(t, server, "POST", "/sites/"+siteID+"/generations", payload, cookie, csrf, "key-1")
	if code != 200 || same["job_id"] != id {
		t.Fatal("idempotencia falló")
	}
	code, _, _ = request(t, server, "GET", "/jobs/"+id, nil, otherCookie, "", "")
	if code != 404 {
		t.Fatalf("job ajeno visible: %d", code)
	}
	item, e := app.claim(context.Background())
	if e != nil || item == nil || item.ID != id {
		t.Fatalf("claim: %v %#v", e, item)
	}
	generated := app.Provider.Create(context.Background(), item.Prompt, app.Config)
	if e = app.finish(context.Background(), item, generated); e != nil {
		t.Fatal(e)
	}
	code, j, _ = request(t, server, "GET", "/jobs/"+id, nil, cookie, "", "")
	if code != 200 || j["status"] != "done" {
		t.Fatalf("job final: %d %#v", code, j)
	}
	article := j["article_id"].(string)
	code, detail, _ := request(t, server, "GET", "/articles/"+article, nil, cookie, "", "")
	if code != 200 {
		t.Fatal(detail)
	}
	slug := detail["slug"].(string)
	code, _, _ = request(t, server, "GET", "/public/sites/"+siteID+"/articles/"+slug, nil, "", "", "")
	if code != 404 {
		t.Fatalf("borrador público: %d", code)
	}
	code, _, _ = request(t, server, "GET", "/articles/"+article, nil, otherCookie, "", "")
	if code != 404 {
		t.Fatalf("artículo ajeno visible: %d", code)
	}
	content := detail["content"].(map[string]any)
	content["intro"] = "<script>evil()</script> <img onerror=evil> javascript:evil()"
	code, _, _ = request(t, server, "PATCH", "/articles/"+article, map[string]any{"title": detail["title"], "slug": slug, "meta_description": detail["meta_description"], "summary": detail["summary"], "content": content}, cookie, csrf, "")
	if code != 200 {
		t.Fatalf("edición: %d", code)
	}
	code, _, _ = request(t, server, "POST", "/articles/"+article+"/approve", nil, cookie, csrf, "")
	if code != 200 {
		t.Fatalf("aprobación: %d", code)
	}
	code, _, _ = request(t, server, "POST", "/articles/"+article+"/publish", nil, cookie, csrf, "")
	if code != 200 {
		t.Fatalf("publicación: %d", code)
	}
	code, public, _ := request(t, server, "GET", "/public/sites/"+siteID+"/articles/"+slug, nil, "", "", "")
	if code != 200 {
		t.Fatalf("publicación pública: %d", code)
	}
	intro := public["content"].(map[string]any)["intro"].(string)
	if strings.ContainsAny(intro, "<>") {
		t.Fatalf("XSS sin sanear: %q", intro)
	}
	var count int
	if e = app.DB.QueryRow(`SELECT count(*) FROM usage_events WHERE tenant_id=$1 AND job_id=$2`, tenant, id).Scan(&count); e != nil || count != 1 {
		t.Fatalf("cobro duplicado: %d %v", count, e)
	}
}
func TestReservaAtomicaYDosIntentos(t *testing.T) {
	app := testApp(t)
	ctx := context.Background()
	var tenant string
	e := app.DB.QueryRow(`INSERT INTO tenants(name,budget_microusd) VALUES($1,100) RETURNING id`, token()).Scan(&tenant)
	if e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan int64, 10)
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, e := app.DB.Exec(`UPDATE tenants SET reserved_microusd=reserved_microusd+60 WHERE id=$1 AND budget_microusd-spent_microusd-reserved_microusd>=60`, tenant)
			if e != nil {
				results <- -1
				return
			}
			n, _ := res.RowsAffected()
			results <- n
		}()
	}
	wg.Wait()
	close(results)
	success := int64(0)
	for n := range results {
		if n < 0 {
			t.Fatal("SQL falló")
		}
		success += n
	}
	if success != 1 {
		t.Fatalf("reservas aceptadas: %d", success)
	}
	var site, job string
	e = app.DB.QueryRow(`INSERT INTO sites(tenant_id,name,domain,canonical_url) VALUES($1,'Test',$2,'http://localhost:18082') RETURNING id`, tenant, token()+".test").Scan(&site)
	if e != nil {
		t.Fatal(e)
	}
	_, e = app.DB.Exec(`UPDATE tenants SET budget_microusd=500000,reserved_microusd=0 WHERE id=$1`, tenant)
	if e != nil {
		t.Fatal(e)
	}
	e = app.DB.QueryRow(`INSERT INTO jobs(tenant_id,site_id,idempotency_key,topic,business,audience) VALUES($1,$2,'retry','Tema','Empresa','Público') RETURNING id`, tenant, site).Scan(&job)
	if e != nil {
		t.Fatal(e)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		item, e := app.claim(ctx)
		if e != nil || item == nil || item.ID != job {
			t.Fatalf("claim intento %d: %v %#v", attempt, e, item)
		}
		if e = app.finish(ctx, item, ProviderResult{Err: errors.New("fallo simulado")}); e != nil {
			t.Fatal(e)
		}
	}
	var status string
	var attempts int
	e = app.DB.QueryRow(`SELECT status,attempts FROM jobs WHERE id=$1`, job).Scan(&status, &attempts)
	if e != nil || status != "failed" || attempts != 2 {
		t.Fatalf("intentos: %s %d %v", status, attempts, e)
	}
	var reserved int64
	e = app.DB.QueryRow(`SELECT reserved_microusd FROM tenants WHERE id=$1`, tenant).Scan(&reserved)
	if e != nil || reserved != 0 {
		t.Fatalf("reserva sin liberar: %d %v", reserved, e)
	}
	_ = time.Second
}
func TestSanitizacion(t *testing.T) {
	c, e := sanitize(Content{Intro: "<script>alert(1)</script>", Sections: []Section{{Heading: "<img onerror=x>", Paragraphs: []string{"javascript:alert(1)"}, Bullets: []string{"<svg onload=x>"}}}, FAQ: []FAQ{{Question: "<b>Pregunta</b>", Answer: "<script>"}}})
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(c)
	if strings.ContainsAny(string(b), "<>") {
		t.Fatal("HTML crudo")
	}
}
