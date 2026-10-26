package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func (a *App) Routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	m.HandleFunc("POST /auth/register", a.register)
	m.HandleFunc("POST /auth/login", a.login)
	m.HandleFunc("POST /auth/logout", a.logout)
	m.HandleFunc("GET /auth/me", a.me)
	m.HandleFunc("GET /sites", a.sites)
	m.HandleFunc("POST /sites", a.createSite)
	m.HandleFunc("PATCH /sites/{id}", a.editSite)
	m.HandleFunc("POST /sites/analyze", a.analyzeSite)
	m.HandleFunc("GET /sites/{id}/research", a.siteResearch)
	m.HandleFunc("POST /sites/{id}/research", a.rereadSite)
	m.HandleFunc("PATCH /sites/{id}/autopilot", a.setAutopilot)
	m.HandleFunc("PATCH /sites/{id}/brand", a.patchBrand)
	m.HandleFunc("POST /sites/{id}/brand/confirm", a.confirmBrand)
	m.HandleFunc("GET /sites/{id}/calendar", a.siteCalendar)
	m.HandleFunc("POST /sites/{id}/verify", a.verifySite)
	m.HandleFunc("DELETE /sites/{id}", a.deleteSite)
	m.HandleFunc("GET /sites/{id}/keywords", a.siteKeywords)
	m.HandleFunc("POST /sites/{id}/google/start", a.googleStart)
	m.HandleFunc("GET /google/callback", a.googleCallback)
	m.HandleFunc("POST /sites/{id}/facebook/start", a.facebookStart)
	m.HandleFunc("GET /facebook/callback", a.facebookCallback)
	m.HandleFunc("GET /sites/{id}/facebook", a.facebookStatus)
	m.HandleFunc("POST /sites/{id}/facebook/page", a.facebookChoose)
	m.HandleFunc("POST /sites/{id}/facebook/share", a.facebookShare)
	m.HandleFunc("DELETE /sites/{id}/facebook", a.facebookDisconnect)
	m.HandleFunc("GET /sites/{id}/analytics", a.siteAnalytics)
	m.HandleFunc("DELETE /sites/{id}/google", a.googleDisconnect)
	m.HandleFunc("POST /sites/{id}/keywords/refresh", a.refreshKeywords)
	m.HandleFunc("POST /sites/{id}/generations", a.generate)
	m.HandleFunc("GET /jobs/{id}", a.job)
	m.HandleFunc("GET /articles", a.articles)
	m.HandleFunc("GET /articles/{id}", a.article)
	m.HandleFunc("PATCH /articles/{id}", a.editArticle)
	m.HandleFunc("POST /articles/{id}/approve", a.approve)
	m.HandleFunc("POST /articles/{id}/publish", a.publish)
	m.HandleFunc("GET /public/sites/{id}", a.publicSite)
	m.HandleFunc("GET /public/domain/{domain}", a.publicDomain)
	m.HandleFunc("GET /public/sites/{id}/articles", a.publicArticles)
	m.HandleFunc("GET /public/sites/{id}/articles/{slug}", a.publicArticle)
	m.HandleFunc("GET /public/sites/{id}/articles/{slug}/image", a.publicArticleImage)
	m.HandleFunc("GET /articles/{id}/image", a.articleImage)
	m.HandleFunc("GET /v1/{site}/posts", a.v1Posts)
	m.HandleFunc("GET /v1/{site}/posts/{slug}", a.v1Post)
	m.HandleFunc("GET /v1/{site}/posts/{slug}/image", a.v1Image)
	m.HandleFunc("GET /api/embed/{id}", a.embed)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		m.ServeHTTP(w, r)
	})
}

type Site struct{ ID, Name, Domain, CanonicalURL, BlogPrefix, BrandDescription, Tone string }

func siteJSON(s Site) map[string]any {
	return map[string]any{"id": s.ID, "name": s.Name, "domain": s.Domain, "canonical_url": s.CanonicalURL, "blog_prefix": s.BlogPrefix, "brand_description": s.BrandDescription, "tone": s.Tone}
}
func scanSite(rows *sql.Rows) (Site, error) {
	var s Site
	e := rows.Scan(&s.ID, &s.Name, &s.Domain, &s.CanonicalURL, &s.BlogPrefix, &s.BrandDescription, &s.Tone)
	return s, e
}
func (a *App) sites(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	rows, e := a.DB.QueryContext(r.Context(), `SELECT id,name,domain,canonical_url,blog_prefix,brand_description,tone FROM sites WHERE tenant_id=$1 ORDER BY created_at DESC`, t)
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows.Close()
	list := []any{}
	for rows.Next() {
		s, e := scanSite(rows)
		if e != nil {
			dbError(w, e)
			return
		}
		list = append(list, siteJSON(s))
	}
	if e = rows.Err(); e != nil {
		dbError(w, e)
		return
	}
	jsonResponse(w, 200, list)
}

type siteInput struct {
	Name             string `json:"name"`
	Domain           string `json:"domain"`
	CanonicalURL     string `json:"canonical_url"`
	BlogPrefix       string `json:"blog_prefix"`
	BrandDescription string `json:"brand_description"`
	Tone             string `json:"tone"`
}

func (a *App) createSite(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	var x siteInput
	if !decode(w, r, &x) {
		return
	}
	if x.BlogPrefix == "" {
		x.BlogPrefix = "/blog"
	}
	u, e := url.Parse(x.CanonicalURL)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || !valid(x.Name, 120) || !valid(x.Domain, 253) || strings.ContainsAny(x.Domain, "/<> ") || !strings.HasPrefix(x.BlogPrefix, "/") || strings.Contains(x.BlogPrefix, "..") || len(x.BlogPrefix) > 100 || len(x.BrandDescription) > 2000 || len(x.Tone) > 500 {
		fail(w, 400, "Datos del sitio inválidos")
		return
	}
	var id string
	e = a.DB.QueryRowContext(r.Context(), `INSERT INTO sites(tenant_id,name,domain,canonical_url,blog_prefix,brand_description,tone) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, t, strings.TrimSpace(x.Name), strings.TrimSpace(x.Domain), strings.TrimRight(u.String(), "/"), strings.TrimRight(x.BlogPrefix, "/"), x.BrandDescription, x.Tone).Scan(&id)
	if e != nil {
		var pg *pgconn.PgError
		if errors.As(e, &pg) && pg.Code == "23505" {
			fail(w, 409, "El dominio ya existe para este cliente")
		} else {
			dbError(w, e)
		}
		return
	}
	jsonResponse(w, 200, map[string]string{"id": id})
}

func (a *App) editSite(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	id, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var x siteInput
	if !decode(w, r, &x) {
		return
	}
	if x.BlogPrefix == "" {
		x.BlogPrefix = "/blog"
	}
	u, e := url.Parse(x.CanonicalURL)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || !valid(x.Name, 120) || !valid(x.Domain, 253) || strings.ContainsAny(x.Domain, "/<> ") || !strings.HasPrefix(x.BlogPrefix, "/") || strings.Contains(x.BlogPrefix, "..") || len(x.BlogPrefix) > 100 || len(x.BrandDescription) > 2000 || len(x.Tone) > 500 {
		fail(w, 400, "Datos del sitio inválidos")
		return
	}
	result, e := a.DB.ExecContext(r.Context(), `UPDATE sites SET name=$1,domain=$2,connected_at=CASE WHEN canonical_url=$3 AND blog_prefix=$4 THEN connected_at END,canonical_url=$3,blog_prefix=$4,brand_description=$5,tone=$6 WHERE id=$7 AND tenant_id=$8`, strings.TrimSpace(x.Name), strings.TrimSpace(x.Domain), strings.TrimRight(u.String(), "/"), strings.TrimRight(x.BlogPrefix, "/"), x.BrandDescription, x.Tone, id, t)
	if e != nil {
		var pg *pgconn.PgError
		if errors.As(e, &pg) && pg.Code == "23505" {
			fail(w, 409, "El dominio ya existe")
		} else {
			dbError(w, e)
		}
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
}

type generationInput struct {
	Topic    string  `json:"topic"`
	Business string  `json:"business"`
	Audience string  `json:"audience"`
	Keyword  *string `json:"keyword"`
}

func (a *App) generate(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	if !a.Limiter.Allow("generate:"+t, 10, time.Hour) {
		fail(w, 429, "Demasiadas generaciones. Intenta más tarde.")
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var x generationInput
	if !decode(w, r, &x) {
		return
	}
	if !valid(x.Topic, 300) || !valid(x.Business, 500) || !valid(x.Audience, 300) || (x.Keyword != nil && len(*x.Keyword) > 120) {
		fail(w, 400, "Datos de generación inválidos")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if !valid(key, 128) {
		fail(w, 400, "Falta Idempotency-Key")
		return
	}
	var exists bool
	e := a.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM sites WHERE id=$1 AND tenant_id=$2)`, site, t).Scan(&exists)
	if e != nil {
		dbError(w, e)
		return
	}
	if !exists {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	var available bool
	e = a.DB.QueryRowContext(r.Context(), `SELECT budget_microusd-spent_microusd-reserved_microusd>0 FROM tenants WHERE id=$1`, t).Scan(&available)
	if e != nil {
		dbError(w, e)
		return
	}
	if !available {
		fail(w, 402, "Presupuesto agotado")
		return
	}
	var id string
	e = a.DB.QueryRowContext(r.Context(), `INSERT INTO jobs(tenant_id,site_id,idempotency_key,topic,business,audience,keyword) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(tenant_id,idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key RETURNING id`, t, site, key, x.Topic, x.Business, x.Audience, x.Keyword).Scan(&id)
	if e != nil {
		dbError(w, e)
		return
	}
	var savedSite, topic, business, audience string
	var keyword sql.NullString
	e = a.DB.QueryRowContext(r.Context(), `SELECT site_id,topic,business,audience,keyword FROM jobs WHERE id=$1 AND tenant_id=$2`, id, t).Scan(&savedSite, &topic, &business, &audience, &keyword)
	if e != nil {
		dbError(w, e)
		return
	}
	sameKeyword := (x.Keyword == nil && !keyword.Valid) || (x.Keyword != nil && keyword.Valid && *x.Keyword == keyword.String)
	if savedSite != site || topic != x.Topic || business != x.Business || audience != x.Audience || !sameKeyword {
		fail(w, 409, "La clave de idempotencia ya corresponde a otra solicitud")
		return
	}
	jsonResponse(w, 200, map[string]string{"job_id": id})
}
func (a *App) job(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	id, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var status string
	var attempts int
	var article, errorMessage sql.NullString
	e := a.DB.QueryRowContext(r.Context(), `SELECT status,attempts,article_id,error_message FROM jobs WHERE id=$1 AND tenant_id=$2`, id, t).Scan(&status, &attempts, &article, &errorMessage)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Job no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	var articleID, errVal any
	if article.Valid {
		articleID = article.String
	}
	if errorMessage.Valid {
		errVal = errorMessage.String
	}
	jsonResponse(w, 200, map[string]any{"status": status, "attempts": attempts, "article_id": articleID, "error": errVal})
}

type Article struct {
	ID, SiteID, Title, Slug, MetaDescription, Summary, Status string
	Content                                                   Content
	PublishedAt                                               *time.Time
	HasImage                                                  bool
}

func articleJSON(a Article) map[string]any {
	return map[string]any{"id": a.ID, "site_id": a.SiteID, "title": a.Title, "slug": a.Slug, "meta_description": a.MetaDescription, "summary": a.Summary, "content": a.Content, "status": a.Status, "published_at": a.PublishedAt, "has_image": a.HasImage}
}
func scanArticle(rows *sql.Rows) (Article, error) {
	var a Article
	var data []byte
	var published sql.NullTime
	e := rows.Scan(&a.ID, &a.SiteID, &a.Title, &a.Slug, &a.MetaDescription, &a.Summary, &data, &a.Status, &published, &a.HasImage)
	if e != nil {
		return a, e
	}
	e = json.Unmarshal(data, &a.Content)
	if published.Valid {
		a.PublishedAt = &published.Time
	}
	return a, e
}

const articleCols = "id,site_id,title,slug,meta_description,summary,content,status,published_at,has_image"

func (a *App) articles(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	rows, e := a.DB.QueryContext(r.Context(), `SELECT id,site_id,title,slug,summary,status,created_at,published_at,has_image FROM articles WHERE tenant_id=$1 ORDER BY created_at DESC`, t)
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows.Close()
	list := []any{}
	for rows.Next() {
		var id, site, title, slug, summary, status string
		var created time.Time
		var published sql.NullTime
		var hasImage bool
		if e = rows.Scan(&id, &site, &title, &slug, &summary, &status, &created, &published, &hasImage); e != nil {
			dbError(w, e)
			return
		}
		var pub any
		if published.Valid {
			pub = published.Time
		}
		list = append(list, map[string]any{"id": id, "site_id": site, "title": title, "slug": slug, "summary": summary, "status": status, "created_at": created, "published_at": pub, "has_image": hasImage})
	}
	jsonResponse(w, 200, list)
}
func (a *App) article(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	id, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	rows, e := a.DB.QueryContext(r.Context(), `SELECT `+articleCols+` FROM articles WHERE id=$1 AND tenant_id=$2`, id, t)
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows.Close()
	if !rows.Next() {
		fail(w, 404, "Artículo no encontrado")
		return
	}
	article, e := scanArticle(rows)
	if e != nil {
		dbError(w, e)
		return
	}
	jsonResponse(w, 200, articleJSON(article))
}

type articleInput struct {
	Title           string  `json:"title"`
	Slug            string  `json:"slug"`
	MetaDescription string  `json:"meta_description"`
	Summary         string  `json:"summary"`
	Content         Content `json:"content"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (a *App) editArticle(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	id, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var x articleInput
	if !decode(w, r, &x) {
		return
	}
	if !valid(x.Title, 180) || !valid(x.Slug, 120) || !slugPattern.MatchString(x.Slug) || !valid(x.MetaDescription, 320) || !valid(x.Summary, 1000) {
		fail(w, 400, "Metadatos inválidos")
		return
	}
	content, e := sanitize(x.Content)
	if e != nil {
		fail(w, 400, "Estructura del artículo inválida")
		return
	}
	body, _ := json.Marshal(content)
	result, e := a.DB.ExecContext(r.Context(), `UPDATE articles SET title=$1,slug=$2,meta_description=$3,summary=$4,content=$5 WHERE id=$6 AND tenant_id=$7 AND status!='published'`, clean(x.Title, 180), x.Slug, clean(x.MetaDescription, 320), clean(x.Summary, 1000), body, id, t)
	if e != nil {
		var pg *pgconn.PgError
		if errors.As(e, &pg) && pg.Code == "23505" {
			fail(w, 409, "El slug ya existe")
		} else {
			dbError(w, e)
		}
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		fail(w, 404, "Artículo no editable")
		return
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
}
func (a *App) approve(w http.ResponseWriter, r *http.Request) {
	a.transition(w, r, "draft", "approved")
}
func (a *App) publish(w http.ResponseWriter, r *http.Request) {
	a.transition(w, r, "approved", "published")
}
func (a *App) transition(w http.ResponseWriter, r *http.Request, from, to string) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	id, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	query := `UPDATE articles SET status=$1 WHERE id=$2 AND tenant_id=$3 AND status=$4`
	if to == "published" {
		query = `UPDATE articles SET status=$1,published_at=now() WHERE id=$2 AND tenant_id=$3 AND status=$4`
	}
	result, e := a.DB.ExecContext(r.Context(), query, to, id, t, from)
	if e != nil {
		dbError(w, e)
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		fail(w, 409, "Estado del artículo inválido")
		return
	}
	jsonResponse(w, 200, map[string]bool{"ok": true})
}
func (a *App) siteBy(w http.ResponseWriter, r *http.Request, query, arg string) {
	var s Site
	e := a.DB.QueryRowContext(r.Context(), query, arg).Scan(&s.ID, &s.Name, &s.Domain, &s.CanonicalURL, &s.BlogPrefix, &s.BrandDescription, &s.Tone)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	jsonResponse(w, 200, siteJSON(s))
}
func (a *App) publicSite(w http.ResponseWriter, r *http.Request) {
	id, ok := requireUUID(w, r.PathValue("id"))
	if ok {
		a.siteBy(w, r, `SELECT id,name,domain,canonical_url,blog_prefix,brand_description,tone FROM sites WHERE id=$1`, id)
	}
}
func (a *App) publicDomain(w http.ResponseWriter, r *http.Request) {
	a.siteBy(w, r, `SELECT id,name,domain,canonical_url,blog_prefix,brand_description,tone FROM sites WHERE domain=$1`, r.PathValue("domain"))
}
func (a *App) publicArticles(w http.ResponseWriter, r *http.Request) {
	id, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	rows, e := a.DB.QueryContext(r.Context(), `SELECT id,title,slug,summary,published_at,has_image FROM articles WHERE site_id=$1 AND status='published' ORDER BY published_at DESC LIMIT 100`, id)
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows.Close()
	list := []any{}
	for rows.Next() {
		var article, title, slug, summary string
		var published time.Time
		var hasImage bool
		if e = rows.Scan(&article, &title, &slug, &summary, &published, &hasImage); e != nil {
			dbError(w, e)
			return
		}
		list = append(list, map[string]any{"id": article, "title": title, "slug": slug, "summary": summary, "published_at": published, "has_image": hasImage})
	}
	jsonResponse(w, 200, list)
}
func (a *App) publicArticle(w http.ResponseWriter, r *http.Request) {
	id, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	rows, e := a.DB.QueryContext(r.Context(), `SELECT `+articleCols+` FROM articles WHERE site_id=$1 AND slug=$2 AND status='published'`, id, r.PathValue("slug"))
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows.Close()
	if !rows.Next() {
		fail(w, 404, "Artículo no encontrado")
		return
	}
	article, e := scanArticle(rows)
	if e != nil {
		dbError(w, e)
		return
	}
	jsonResponse(w, 200, articleJSON(article))
}
func (a *App) embed(w http.ResponseWriter, r *http.Request) {
	id, ok := requireUUID(w, strings.TrimSuffix(r.PathValue("id"), ".js"))
	if !ok {
		return
	}
	var canonical, prefix, domain string
	var connected bool
	e := a.DB.QueryRowContext(r.Context(), `SELECT canonical_url,blog_prefix,domain,connected_at IS NOT NULL FROM sites WHERE id=$1`, id).Scan(&canonical, &prefix, &domain, &connected)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	rows, e := a.DB.QueryContext(r.Context(), `SELECT title,slug,summary,published_at FROM articles WHERE site_id=$1 AND status='published' ORDER BY published_at DESC LIMIT 100`, id)
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var title, slug, summary string
		var published time.Time
		if e = rows.Scan(&title, &slug, &summary, &published); e != nil {
			dbError(w, e)
			return
		}
		items = append(items, map[string]any{"title": title, "slug": slug, "summary": summary, "published_at": published})
	}
	data, _ := json.Marshal(items)
	base, _ := json.Marshal(blogBase(connected, canonical, prefix, domain) + "/")
	script := fmt.Sprintf(`(()=>{const el=document.getElementById('faro-blog');if(!el)return;const root=el.attachShadow({mode:'open'});const style=document.createElement('style');style.textContent=':host{display:block;font:inherit}article{padding:1rem 0;border-bottom:1px solid #ddd}a{color:inherit;font-weight:700}p{margin:.4rem 0}';root.append(style);const base=%s;for(const a of %s){const row=document.createElement('article');const link=document.createElement('a');link.href=base+encodeURIComponent(a.slug);link.textContent=a.title;const desc=document.createElement('p');desc.textContent=a.summary;const date=document.createElement('time');date.dateTime=a.published_at;date.textContent=new Date(a.published_at).toLocaleDateString('es-MX');row.append(link,desc,date);root.append(row)}})();`, base, data)
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(script))
}
