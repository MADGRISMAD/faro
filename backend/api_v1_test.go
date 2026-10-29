package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIv1EntregaSoloDatos(t *testing.T) {
	minAutoWords = 0
	defer func() { minAutoWords = 500 }()
	app := testApp(t)
	app.Config.ImageModel, app.Config.ImageRate = "mock", 30000
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()
	cookie, csrf, tenant := registerTest(t, srv)
	domain := "datos-" + token()[:8] + ".test"
	_, res, _ := request(t, srv, "POST", "/sites/analyze", map[string]string{"url": "www." + domain}, cookie, csrf, "")
	siteID := res["site_id"].(string)
	app.DB.Exec(`UPDATE jobs SET status='failed' WHERE site_id=$1`, siteID)
	app.DB.Exec(`UPDATE sites SET autopilot=true WHERE id=$1`, siteID)
	take := func(kind string) *WorkItem {
		for i := 0; i < 40; i++ {
			j, e := app.claim(context.Background())
			if e != nil || j == nil {
				break
			}
			if j.Tenant == tenant && j.Kind == kind {
				return j
			}
		}
		t.Fatalf("sin job %s", kind)
		return nil
	}
	var jobID string
	app.DB.QueryRow(`INSERT INTO jobs(tenant_id,site_id,idempotency_key,topic,business,audience,kind) VALUES($1,$2,$3,'Tema','Negocio','Público','article') RETURNING id`, tenant, siteID, token()).Scan(&jobID)
	j := take("article")
	if e := app.finish(context.Background(), j, app.Provider.Create(context.Background(), j.Prompt, app.Config)); e != nil {
		t.Fatal(e)
	}
	app.runImage(context.Background(), take("image"))
	var slug string
	app.DB.QueryRow(`SELECT slug FROM articles WHERE site_id=$1`, siteID).Scan(&slug)

	for _, ref := range []string{siteID, domain, "WWW." + strings.ToUpper(domain)} { // UUID, dominio y dominio con www/mayúsculas
		code, list, _ := request(t, srv, "GET", "/v1/"+ref+"/posts", nil, "", "", "")
		posts, _ := list["posts"].([]any)
		if code != 200 || len(posts) != 1 || list["total"].(float64) != 1 {
			t.Fatalf("lista por %s: %d %#v", ref, code, list)
		}
	}
	_, list, _ := request(t, srv, "GET", "/v1/"+domain+"/posts", nil, "", "", "")
	first := list["posts"].([]any)[0].(map[string]any)
	img := first["image"].(map[string]any)
	if rm, _ := first["reading_minutes"].(float64); rm < 1 {
		t.Fatalf("minutos de lectura: %#v", first)
	}
	if first["path"] != "/blog/"+slug || !strings.HasSuffix(img["url"].(string), "/api/v1/"+domain+"/posts/"+slug+"/image") || img["alt"] == "" {
		t.Fatalf("post: %#v", first)
	}
	code, post, _ := request(t, srv, "GET", "/v1/"+domain+"/posts/"+slug, nil, "", "", "")
	ld := post["jsonld"].([]any)
	if code != 200 || len(ld) != 3 || ld[0].(map[string]any)["@type"] != "BlogPosting" || ld[1].(map[string]any)["@type"] != "BreadcrumbList" || ld[2].(map[string]any)["@type"] != "FAQPage" {
		t.Fatalf("artículo: %d %#v", code, post["jsonld"])
	}
	// una dirección anterior sigue resolviendo y responde con la dirección actual
	app.DB.Exec(`INSERT INTO article_slug_aliases(site_id,old_slug,article_id) SELECT site_id,'direccion-anterior',id FROM articles WHERE site_id=(SELECT id FROM sites WHERE domain=$1) AND slug=$2`, domain, slug)
	if code, old, _ := request(t, srv, "GET", "/v1/"+domain+"/posts/direccion-anterior", nil, "", "", ""); code != 200 || old["slug"] != slug {
		t.Fatalf("alias: %d %v", code, old["slug"])
	}
	if rm, _ := post["reading_minutes"].(float64); rm < 1 {
		t.Fatalf("minutos de lectura del artículo: %#v", post["reading_minutes"])
	}
	if !strings.HasSuffix(post["canonical"].(string), "/blog/"+slug) || post["seo"].(map[string]any)["description"] == "" || !strings.Contains(post["html"].(string), "<h2>") {
		t.Fatalf("seo/html: %#v", post)
	}
	req, _ := http.NewRequest("GET", srv.URL+"/v1/"+domain+"/posts/"+slug+"/image", nil)
	if r, e := http.DefaultClient.Do(req); e != nil || r.StatusCode != 200 || r.Header.Get("Content-Type") != "image/jpeg" || r.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("imagen: %v %v", e, r)
	}
	// el cuerpo se escapa aunque llegara algo raro a la base
	app.DB.Exec(`UPDATE articles SET content=jsonb_set(content,'{intro}','"<img src=x onerror=alert(1)>"') WHERE site_id=$1`, siteID)
	_, post, _ = request(t, srv, "GET", "/v1/"+domain+"/posts/"+slug, nil, "", "", "")
	if h := post["html"].(string); strings.Contains(h, "<img") || !strings.Contains(h, "&lt;img") {
		t.Fatalf("html sin escapar: %s", h)
	}
	if code, _, _ := request(t, srv, "GET", "/v1/no-existe.test/posts", nil, "", "", ""); code != 404 {
		t.Fatalf("sitio inexistente: %d", code)
	}
	// un borrador nunca sale
	app.DB.Exec(`UPDATE articles SET status='draft' WHERE site_id=$1`, siteID)
	if code, _, _ := request(t, srv, "GET", "/v1/"+domain+"/posts/"+slug, nil, "", "", ""); code != 404 {
		t.Fatalf("borrador visible: %d", code)
	}
	if _, l, _ := request(t, srv, "GET", "/v1/"+domain+"/posts", nil, "", "", ""); l["total"].(float64) != 0 {
		t.Fatalf("borrador en la lista: %#v", l)
	}
	app.DB.Exec(`UPDATE jobs SET status='failed' WHERE tenant_id=$1 AND status IN ('queued','running')`, tenant)
}
