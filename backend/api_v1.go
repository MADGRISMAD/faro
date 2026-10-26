package main

// API pública de lectura: Faro entrega los DATOS (artículos, imagen, SEO listo) y el sitio del cliente pone su propio diseño.
//   GET /v1/{sitio}/posts?limit=20&page=1      lista
//   GET /v1/{sitio}/posts/{slug}               artículo completo (estructura, html, jsonld)
//   GET /v1/{sitio}/posts/{slug}/image         portada (JPEG)
// {sitio} es el UUID o el dominio (con o sin www). Solo expone artículos publicados. CORS abierto: es contenido público.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type v1Site struct{ ID, Name, Domain, Canonical, Prefix string }

func (a *App) v1Site(r *http.Request) (v1Site, error) {
	ref := strings.ToLower(strings.TrimSpace(r.PathValue("site")))
	var s v1Site
	q := `SELECT id,name,domain,canonical_url,blog_prefix FROM sites WHERE domain=$1`
	if len(ref) == 36 {
		q = `SELECT id,name,domain,canonical_url,blog_prefix FROM sites WHERE id=$1`
	} else {
		ref = strings.TrimPrefix(ref, "www.")
	}
	return s, a.DB.QueryRowContext(r.Context(), q, ref).Scan(&s.ID, &s.Name, &s.Domain, &s.Canonical, &s.Prefix)
}

func v1Headers(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
}

func imageURL(s v1Site, slug string) string {
	return strings.TrimRight(env("PUBLIC_BASE_URL", ""), "/") + "/api/v1/" + s.Domain + "/posts/" + slug + "/image"
}

func v1Image(s v1Site, slug, title string, has bool) any {
	if !has {
		return nil
	}
	return map[string]any{"url": imageURL(s, slug), "alt": "Portada: " + title, "width": 1376, "height": 768}
}

func (a *App) v1Posts(w http.ResponseWriter, r *http.Request) {
	s, e := a.v1Site(r)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	var total int
	_ = a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM articles WHERE site_id=$1 AND status='published'`, s.ID).Scan(&total)
	rows, e := a.DB.QueryContext(r.Context(), `SELECT id,slug,title,summary,meta_description,published_at,has_image FROM articles WHERE site_id=$1 AND status='published' ORDER BY published_at DESC,id LIMIT $2 OFFSET $3`, s.ID, limit, (page-1)*limit)
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows.Close()
	posts := []any{}
	for rows.Next() {
		var id, slug, title, summary, meta string
		var at time.Time
		var img bool
		if e = rows.Scan(&id, &slug, &title, &summary, &meta, &at, &img); e != nil {
			dbError(w, e)
			return
		}
		posts = append(posts, map[string]any{"id": id, "slug": slug, "title": title, "summary": summary, "meta_description": meta, "published_at": at, "path": s.Prefix + "/" + slug, "image": v1Image(s, slug, title, img)})
	}
	v1Headers(w)
	jsonResponse(w, 200, map[string]any{"site": map[string]any{"id": s.ID, "name": s.Name, "domain": s.Domain, "canonical_url": s.Canonical, "blog_prefix": s.Prefix, "language": "es-MX"}, "page": page, "limit": limit, "total": total, "posts": posts})
}

// contentHTML: el cuerpo ya viene saneado (sin < ni >); aun así se escapa por seguridad en profundidad.
func contentHTML(c Content) string {
	var b strings.Builder
	p := func(tag, s string) { fmt.Fprintf(&b, "<%s>%s</%s>", tag, html.EscapeString(s), tag) }
	p("p", c.Intro)
	for _, s := range c.Sections {
		p("h2", s.Heading)
		for _, x := range s.Paragraphs {
			p("p", x)
		}
		if len(s.Bullets) > 0 {
			b.WriteString("<ul>")
			for _, x := range s.Bullets {
				p("li", x)
			}
			b.WriteString("</ul>")
		}
	}
	if len(c.FAQ) > 0 {
		p("h2", "Preguntas frecuentes")
		for _, f := range c.FAQ {
			p("h3", f.Question)
			p("p", f.Answer)
		}
	}
	return b.String()
}

func jsonLD(s v1Site, title, meta, slug string, at time.Time, img bool, c Content) []any {
	url := strings.TrimRight(s.Canonical, "/") + s.Prefix + "/" + slug
	post := map[string]any{"@context": "https://schema.org", "@type": "BlogPosting", "headline": title, "description": meta, "datePublished": at, "dateModified": at, "mainEntityOfPage": url, "publisher": map[string]any{"@type": "Organization", "name": s.Name}}
	if img {
		post["image"] = imageURL(s, slug)
	}
	out := []any{post}
	if len(c.FAQ) > 0 {
		qs := []any{}
		for _, f := range c.FAQ {
			qs = append(qs, map[string]any{"@type": "Question", "name": f.Question, "acceptedAnswer": map[string]any{"@type": "Answer", "text": f.Answer}})
		}
		out = append(out, map[string]any{"@context": "https://schema.org", "@type": "FAQPage", "mainEntity": qs})
	}
	return out
}

func (a *App) v1Post(w http.ResponseWriter, r *http.Request) {
	s, e := a.v1Site(r)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	var id, slug, title, summary, meta string
	var data []byte
	var at time.Time
	var img bool
	e = a.DB.QueryRowContext(r.Context(), `SELECT id,slug,title,summary,meta_description,content,published_at,has_image FROM articles WHERE site_id=$1 AND slug=$2 AND status='published'`, s.ID, r.PathValue("slug")).Scan(&id, &slug, &title, &summary, &meta, &data, &at, &img)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Artículo no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	var c Content
	if json.Unmarshal(data, &c) != nil {
		dbError(w, errors.New("contenido corrupto"))
		return
	}
	v1Headers(w)
	jsonResponse(w, 200, map[string]any{
		"id": id, "site_id": s.ID, "slug": slug, "title": title, "summary": summary, "published_at": at,
		"path": s.Prefix + "/" + slug, "canonical": strings.TrimRight(s.Canonical, "/") + s.Prefix + "/" + slug,
		"image":   v1Image(s, slug, title, img),
		"seo":     map[string]any{"title": title + " | " + s.Name, "description": meta},
		"content": c, "html": contentHTML(c), "jsonld": jsonLD(s, title, meta, slug, at, img, c),
	})
}

func (a *App) v1Image(w http.ResponseWriter, r *http.Request) {
	s, e := a.v1Site(r)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	serveImage(w, r, a.DB, `SELECT i.mime,i.data,i.alt FROM article_images i JOIN articles a ON a.id=i.article_id WHERE a.site_id=$1 AND a.slug=$2 AND a.status='published'`, false, s.ID, r.PathValue("slug"))
}
