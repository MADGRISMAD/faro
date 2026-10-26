package main

// Investigación automática: el usuario solo da la URL de su sitio. Un job de tipo "research" lee el sitio
// (con protección SSRF), Gemini deduce negocio, público, tono y temas, y el autopiloto empieza a escribir solo.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Topic struct {
	Title   string `json:"title"`
	Keyword string `json:"keyword"`
	Intent  string `json:"intent"`
	Why     string `json:"why"`
}
type Profile struct {
	Name      string   `json:"name"`
	Business  string   `json:"business"`
	Audience  string   `json:"audience"`
	Tone      string   `json:"tone"`
	Location  string   `json:"location"`
	Offerings []string `json:"offerings"`
	Topics    []Topic  `json:"topics"`
	// Sensitive: salud, legal o finanzas. Google exige más cuidado (experiencia y revisión humana), así que nada se publica solo.
	Sensitive       bool   `json:"sensitive"`
	SensitiveReason string `json:"sensitive_reason"`
}
type ResearchInput struct{ URL, Text string }
type ResearchResult struct {
	Profile                   Profile
	InputTokens, OutputTokens int
	Unknown                   bool
	Err                       error
}

// ───────────── URL: acepta "www.x.com", "x.com/blog", "http://x.com"… y se queda con el origen ─────────────

func normalizeURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 300 || strings.ContainsAny(raw, " <>\"'\x00") {
		return nil, errors.New("Escribe la dirección de tu sitio, por ejemplo tusitio.com")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + strings.TrimPrefix(raw, "//")
	}
	u, e := url.Parse(raw)
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if e != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || !strings.Contains(host, ".") || net.ParseIP(host) != nil {
		return nil, errors.New("Esa dirección no parece un sitio web. Usa un dominio como tusitio.com")
	}
	if p := u.Port(); p != "" && p != "80" && p != "443" {
		return nil, errors.New("Solo se aceptan sitios en los puertos web normales")
	}
	return &url.URL{Scheme: u.Scheme, Host: host}, nil
}

// ───────────── Lectura segura del sitio (SSRF) ─────────────

var allowPrivate = false // solo las pruebas lo activan, para leer un servidor local

var cgnat = mustCIDR("100.64.0.0/10")

func mustCIDR(s string) *net.IPNet { _, n, _ := net.ParseCIDR(s); return n }
func publicIP(ip net.IP) bool {
	return !(ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() || cgnat.Contains(ip))
}

func safeClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	tr := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, e := net.SplitHostPort(addr)
			if e != nil {
				return nil, e
			}
			if !allowPrivate && port != "80" && port != "443" {
				return nil, errors.New("puerto no permitido")
			}
			ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
			if e != nil || len(ips) == 0 {
				return nil, errors.New("no se pudo resolver el dominio")
			}
			for _, ip := range ips { // si UNA dirección es privada se rechaza todo: evita DNS con respuestas mezcladas
				if !allowPrivate && !publicIP(ip.IP) {
					return nil, errors.New("dirección no permitida")
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
		TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 8 * time.Second, DisableKeepAlives: true,
	}
	return &http.Client{Transport: tr, Timeout: 10 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 4 {
			return errors.New("demasiadas redirecciones")
		}
		return nil
	}}
}

// fetchPage descarga una página HTML (máx. 1.5 MB). Devuelve la URL final tras redirecciones.
func fetchPage(ctx context.Context, c *http.Client, u string) (string, *url.URL, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", u, nil)
	if e != nil {
		return "", nil, e
	}
	req.Header.Set("User-Agent", "FaroContenidoBot/1.0 (+lectura de tu sitio para planear contenido)")
	req.Header.Set("Accept", "text/html")
	res, e := c.Do(req)
	if e != nil {
		return "", nil, e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", nil, fmt.Errorf("el sitio respondió con estado %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "" && !strings.Contains(ct, "html") {
		return "", nil, errors.New("no es una página web")
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, 1500000))
	return string(b), res.Request.URL, e
}

var (
	reDrop  = regexp.MustCompile(`(?is)<(script|style|noscript|svg|template|iframe)\b.*?</\s*(script|style|noscript|svg|template|iframe)\s*>|<!--.*?-->`)
	reTag   = regexp.MustCompile(`(?s)<[^>]*>`)
	reSpace = regexp.MustCompile(`\s+`)
	reTitle = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reMeta  = regexp.MustCompile(`(?is)<meta\s+[^>]*(?:name|property)=["'](?:description|og:description)["'][^>]*content=["']([^"']*)["']`)
	reHead  = regexp.MustCompile(`(?is)<h[1-3][^>]*>(.*?)</h[1-3]>`)
	reHref  = regexp.MustCompile(`(?is)<a\s[^>]*href=["']([^"'#]+)["']`)
	reAsset = regexp.MustCompile(`(?i)\.(jpe?g|png|gif|webp|svg|pdf|zip|css|js|ico|mp4|mp3|xml|json)(\?|$)`)
)

func textOf(s string) string {
	s = reTag.ReplaceAllString(s, " ")
	s = strings.NewReplacer("&amp;", "&", "&nbsp;", " ", "&quot;", "\"", "&#39;", "'", "&lt;", "<", "&gt;", ">").Replace(s)
	return strings.TrimSpace(reSpace.ReplaceAllString(s, " "))
}

// extract resume una página: título, descripción, encabezados y texto visible.
func extract(html string) string {
	var b strings.Builder
	if m := reTitle.FindStringSubmatch(html); m != nil {
		b.WriteString("Título: " + textOf(m[1]) + "\n")
	}
	if m := reMeta.FindStringSubmatch(html); m != nil {
		b.WriteString("Descripción: " + textOf(m[1]) + "\n")
	}
	html = reDrop.ReplaceAllString(html, " ")
	n := 0
	for _, m := range reHead.FindAllStringSubmatch(html, 25) {
		if t := textOf(m[1]); t != "" {
			b.WriteString("# " + t + "\n")
			n++
		}
	}
	body := textOf(html)
	if len(body) > 3500 {
		body = body[:3500]
	}
	b.WriteString(body + "\n")
	return b.String()
}

// internalLinks elige hasta n páginas del mismo sitio, empezando por las de ruta más corta (servicios, nosotros, precios…).
func internalLinks(html string, base *url.URL, n int) []string {
	seen := map[string]bool{base.String(): true, strings.TrimSuffix(base.String(), "/") + "/": true}
	var out []string
	for _, m := range reHref.FindAllStringSubmatch(html, 200) {
		l, e := base.Parse(strings.TrimSpace(m[1]))
		if e != nil || (l.Scheme != "http" && l.Scheme != "https") || !strings.EqualFold(strings.TrimPrefix(l.Hostname(), "www."), strings.TrimPrefix(base.Hostname(), "www.")) || reAsset.MatchString(l.Path) || l.Path == "" || l.Path == "/" {
			continue
		}
		l.RawQuery, l.Fragment = "", ""
		if k := l.String(); !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	// ruta corta primero: suelen ser las secciones principales
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && len(out[j]) < len(out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func readSite(ctx context.Context, start string) (text string, final *url.URL, err error) {
	c := safeClient()
	html, final, e := fetchPage(ctx, c, start)
	if e != nil {
		return "", nil, fmt.Errorf("No pude abrir tu sitio (%s). Revisa que la dirección sea correcta y pública", e.Error())
	}
	var b strings.Builder
	b.WriteString("PÁGINA PRINCIPAL " + final.String() + "\n" + extract(html))
	for _, l := range internalLinks(html, final, 4) {
		if h, _, e := fetchPage(ctx, c, l); e == nil {
			b.WriteString("\nPÁGINA " + l + "\n" + extract(h))
		}
		if b.Len() > 14000 {
			break
		}
	}
	return b.String(), final, nil
}

// ───────────── Proveedores ─────────────

func (*Mock) Research(_ context.Context, in ResearchInput, _ Config) ResearchResult {
	name := "Tu negocio"
	for _, l := range strings.Split(in.Text, "\n") {
		if strings.HasPrefix(l, "Título: ") {
			name = strings.TrimSpace(strings.SplitN(strings.TrimPrefix(l, "Título: "), "|", 2)[0])
			break
		}
	}
	t := func(title, kw, intent string) Topic {
		return Topic{title, kw, intent, "Responde una duda frecuente de tus clientes."}
	}
	return ResearchResult{Profile: Profile{Name: name, Business: "Negocio detectado en " + in.URL, Audience: "Clientes potenciales en México", Tone: "Cercano y profesional", Location: "México",
		Offerings: []string{"Servicio principal"}, Topics: []Topic{t("Cómo elegir "+name, "elegir "+name, "comparison"), t("Preguntas frecuentes sobre "+name, "preguntas frecuentes", "informational"), t("Precios de "+name+": qué incluye", "precios", "purchase"), t("Errores comunes al contratar "+name, "errores comunes", "informational")}}, InputTokens: 300, OutputTokens: 400}
}

func (g *Gemini) Research(ctx context.Context, in ResearchInput, c Config) ResearchResult {
	str := map[string]any{"type": "STRING"}
	list := func(item any) map[string]any { return map[string]any{"type": "ARRAY", "items": item} }
	topic := map[string]any{"type": "OBJECT", "properties": map[string]any{"title": str, "keyword": str, "intent": map[string]any{"type": "STRING", "enum": []string{"informational", "comparison", "purchase", "local"}}, "why": str}, "required": []string{"title", "keyword", "intent", "why"}}
	schema := map[string]any{"type": "OBJECT", "properties": map[string]any{"name": str, "business": str, "audience": str, "tone": str, "location": str, "offerings": list(str), "topics": list(topic), "sensitive": map[string]any{"type": "BOOLEAN"}, "sensitive_reason": str}, "required": []string{"name", "business", "audience", "tone", "location", "offerings", "topics", "sensitive", "sensitive_reason"}}
	prompt := "Eres estratega de contenido y SEO. Lee el contenido de este sitio web (texto extraído, puede estar incompleto) y deduce lo siguiente SIN inventar datos que no estén en el sitio: name (nombre del negocio), business (qué hace, en 1-2 frases), audience (a quién le vende), tone (tono de voz de la marca, 3-6 palabras), location (ciudad o región donde opera, o \"México\" si no se sabe), offerings (hasta 8 productos o servicios reales). Después propón EXACTAMENTE 12 temas de artículos de blog en español de México que traigan visitas de Google a este negocio: mezcla intención informativa, de comparación, de compra y local; cada uno con title (título atractivo), keyword (palabra clave principal natural, sin inventar volúmenes), intent y why (por qué le sirve a este negocio, 1 frase). No repitas temas ni propongas nada que el negocio no pueda atender. Además marca sensitive=true si el negocio es de salud, medicina, psicología, derecho, finanzas, seguros o inversiones (temas donde Google exige experiencia y revisión de un profesional) y explica en sensitive_reason en una frase; si no, false y vacío.\n\nSITIO: " + in.URL + "\n\n" + in.Text
	text, inT, outT, unknown, e := geminiJSON(ctx, g.Client, c, prompt, schema, 3000)
	r := ResearchResult{InputTokens: inT, OutputTokens: outT, Unknown: unknown, Err: e}
	if e == nil && json.Unmarshal([]byte(text), &r.Profile) != nil {
		r.Err = errors.New("Gemini devolvió JSON inválido")
	}
	return r
}

// geminiJSON hace una llamada con salida JSON con esquema y devuelve texto + tokens usados.
func geminiJSON(ctx context.Context, cl *http.Client, c Config, prompt string, schema map[string]any, maxOut int) (string, int, int, bool, error) {
	body, _ := json.Marshal(map[string]any{"contents": []any{map[string]any{"parts": []any{map[string]string{"text": prompt}}}}, "generationConfig": map[string]any{"responseMimeType": "application/json", "responseSchema": schema, "maxOutputTokens": maxOut, "temperature": 0.4, "thinkingConfig": map[string]any{"thinkingLevel": "low"}}})
	req, e := http.NewRequestWithContext(ctx, "POST", "https://generativelanguage.googleapis.com/v1beta/models/"+c.Model+":generateContent", bytes.NewReader(body))
	if e != nil {
		return "", 0, 0, false, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.Key)
	res, e := cl.Do(req)
	if e != nil {
		return "", 0, 0, true, errors.New("No se pudo contactar a Gemini")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", 0, 0, false, fmt.Errorf("Gemini respondió con estado %d", res.StatusCode)
	}
	data, e := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var wire struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	var raw struct {
		U struct {
			P int `json:"promptTokenCount"`
			C int `json:"candidatesTokenCount"`
			T int `json:"thoughtsTokenCount"`
		} `json:"usageMetadata"`
	}
	if e != nil || json.Unmarshal(data, &wire) != nil || json.Unmarshal(data, &raw) != nil {
		return "", 0, 0, false, errors.New("Respuesta Gemini inválida")
	}
	in, out := raw.U.P, raw.U.C+raw.U.T
	if len(wire.Candidates) == 0 || len(wire.Candidates[0].Content.Parts) == 0 {
		return "", in, out, false, errors.New("Gemini no devolvió texto")
	}
	return wire.Candidates[0].Content.Parts[0].Text, in, out, false, nil
}

func cleanProfile(p Profile) (Profile, error) {
	p.Name, p.Business, p.Audience = clean(strings.TrimSpace(p.Name), 120), clean(strings.TrimSpace(p.Business), 500), clean(strings.TrimSpace(p.Audience), 300)
	p.Tone, p.Location = clean(strings.TrimSpace(p.Tone), 120), clean(strings.TrimSpace(p.Location), 120)
	p.SensitiveReason = clean(strings.TrimSpace(p.SensitiveReason), 200)
	if !valid(p.Name, 120) || !valid(p.Business, 500) || !valid(p.Audience, 300) {
		return p, errors.New("perfil incompleto")
	}
	offers := []string{}
	for _, o := range p.Offerings {
		if o = clean(strings.TrimSpace(o), 100); o != "" && len(offers) < 8 {
			offers = append(offers, o)
		}
	}
	p.Offerings = offers
	topics, seen := []Topic{}, map[string]bool{}
	for _, t := range p.Topics {
		t.Title, t.Keyword, t.Why = clean(strings.TrimSpace(t.Title), 200), clean(strings.TrimSpace(t.Keyword), 100), clean(strings.TrimSpace(t.Why), 300)
		if t.Intent != "informational" && t.Intent != "comparison" && t.Intent != "purchase" && t.Intent != "local" {
			t.Intent = "informational"
		}
		if k := strings.ToLower(t.Title); t.Title != "" && !seen[k] && len(topics) < 20 {
			seen[k] = true
			topics = append(topics, t)
		}
	}
	if len(topics) < 3 {
		return p, errors.New("muy pocos temas")
	}
	p.Topics = topics
	return p, nil
}

// ───────────── Worker: job de investigación ─────────────

func (a *App) runResearch(ctx context.Context, j *WorkItem) {
	callCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	var r ResearchResult
	var fetchErr error
	text, final, e := readSite(callCtx, j.Prompt.Topic)
	if e != nil {
		fetchErr = e
	} else {
		r = a.Provider.Research(callCtx, ResearchInput{URL: final.String(), Text: text}, a.Config)
		if r.Err == nil {
			r.Profile, r.Err = cleanProfile(r.Profile)
			if r.Err != nil {
				r.Err = errors.New("Gemini devolvió un perfil inválido")
			}
		}
	}
	if e := a.finishResearch(ctx, j, r, fetchErr, final); e != nil {
		slog.Error("No se pudo completar la investigación", "error", e)
	}
}

func (a *App) finishResearch(ctx context.Context, j *WorkItem, r ResearchResult, fetchErr error, final *url.URL) error {
	tx, e := a.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var status string
	var attempt int
	if e = tx.QueryRowContext(ctx, `SELECT status,attempts FROM jobs WHERE id=$1 FOR UPDATE`, j.ID).Scan(&status, &attempt); e != nil {
		return e
	}
	if status != "running" || attempt != j.Attempt {
		return errors.New("job cambió de estado durante la llamada")
	}
	cost, usage := actualCost(a.Config, r.InputTokens, r.OutputTokens), "measured"
	if r.Err != nil && r.Unknown && r.InputTokens == 0 && r.OutputTokens == 0 {
		cost, usage = j.Reserve, "unknown"
	}
	if cost > j.Reserve {
		return errors.New("uso mayor que reserva: requiere conciliación")
	}
	if _, e = tx.ExecContext(ctx, `UPDATE tenants SET reserved_microusd=reserved_microusd-$1,spent_microusd=spent_microusd+$2 WHERE id=$3`, j.Reserve, cost, j.Tenant); e != nil {
		return e
	}
	if cost > 0 {
		if _, e = tx.ExecContext(ctx, `INSERT INTO usage_events(tenant_id,job_id,attempt,model,input_tokens,output_tokens,cost_microusd,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, j.Tenant, j.ID, j.Attempt, a.Config.Model, r.InputTokens, r.OutputTokens, cost, usage); e != nil {
			return e
		}
	}
	if fetchErr != nil || r.Err != nil {
		next, msg := "queued", ""
		if fetchErr != nil {
			next, msg = "failed", fetchErr.Error() // un sitio que no abre no mejora reintentando
		} else {
			msg = r.Err.Error()
			if j.Attempt >= 2 {
				next = "failed"
			}
		}
		if _, e = tx.ExecContext(ctx, `UPDATE jobs SET status=$1,reserved_microusd=0,error_message=$2,updated_at=now() WHERE id=$3`, next, msg, j.ID); e != nil {
			return e
		}
		return tx.Commit()
	}
	p := r.Profile
	blob, _ := json.Marshal(p)
	brand := p.Business
	if len(p.Offerings) > 0 {
		brand += " Ofrece: " + strings.Join(p.Offerings, ", ") + "."
	}
	canonical := ""
	if final != nil {
		canonical = final.Scheme + "://" + final.Host
	}
	if _, e = tx.ExecContext(ctx, `UPDATE sites SET name=$1,brand_description=$2,tone=$3,research=$4,canonical_url=CASE WHEN $5<>'' AND replace(replace(lower($5),'https://',''),'http://','') IN (domain,'www.'||domain) THEN $5 ELSE canonical_url END WHERE id=$6`, p.Name, clean(brand, 1500), p.Tone, blob, canonical, j.Site); e != nil {
		return e
	}
	var auto bool
	if e = tx.QueryRowContext(ctx, `SELECT autopilot FROM sites WHERE id=$1`, j.Site).Scan(&auto); e != nil {
		return e
	}
	if p.Sensitive { // sector de salud, legal o finanzas: todo queda en borrador para revisión humana
		if _, e = tx.ExecContext(ctx, `UPDATE sites SET auto_publish=false WHERE id=$1`, j.Site); e != nil {
			return e
		}
	}
	if auto { // un sitio nuevo arranca con UN artículo; el resto sigue el ritmo (ver autopilotTick)
		if e = enqueueTopic(ctx, tx, j.Tenant, j.Site, p, p.Topics[0]); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, `UPDATE jobs SET status='done',reserved_microusd=0,error_message=NULL,updated_at=now() WHERE id=$1`, j.ID); e != nil {
		return e
	}
	return tx.Commit()
}

func topicKey(site, title string) string {
	h := sha256.Sum256([]byte(site + "\x00" + strings.ToLower(title)))
	return "auto-" + hex.EncodeToString(h[:8])
}

func enqueueTopic(ctx context.Context, tx *sql.Tx, tenant, site string, p Profile, t Topic) error {
	kw := sql.NullString{String: t.Keyword, Valid: t.Keyword != ""}
	_, e := tx.ExecContext(ctx, `INSERT INTO jobs(tenant_id,site_id,idempotency_key,topic,business,audience,keyword) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(tenant_id,idempotency_key) DO NOTHING`, tenant, site, topicKey(site, t.Title), t.Title, p.Business, p.Audience, kw)
	return e
}

// autopilotTick: por cada sitio con autopiloto, mantiene el ritmo semanal con el siguiente tema sin usar.
func (a *App) autopilotTick(ctx context.Context) error {
	rows, e := a.DB.QueryContext(ctx, `SELECT s.id,s.tenant_id,s.posts_per_week,s.research FROM sites s WHERE s.autopilot AND s.research IS NOT NULL AND
		(SELECT count(*) FROM jobs j WHERE j.site_id=s.id AND j.kind='article' AND j.created_at>now()-interval '7 days')<LEAST(s.posts_per_week,CASE WHEN s.created_at>now()-interval '14 days' THEN 1 ELSE 3 END) AND
		NOT EXISTS(SELECT 1 FROM jobs j WHERE j.site_id=s.id AND j.kind='article' AND (j.status IN ('queued','running') OR j.created_at>now()-make_interval(hours => 168/LEAST(s.posts_per_week,CASE WHEN s.created_at>now()-interval '14 days' THEN 1 ELSE 3 END))))`)
	if e != nil {
		return e
	}
	type due struct {
		site, tenant string
		blob         []byte
	}
	var list []due
	for rows.Next() {
		var d due
		var n int
		if e = rows.Scan(&d.site, &d.tenant, &n, &d.blob); e != nil {
			rows.Close()
			return e
		}
		list = append(list, d)
	}
	rows.Close()
	for _, d := range list {
		var p Profile
		if json.Unmarshal(d.blob, &p) != nil {
			continue
		}
		var ok bool
		if e = a.DB.QueryRowContext(ctx, `SELECT budget_microusd-spent_microusd-reserved_microusd>=$1 FROM tenants WHERE id=$2`, reserveAmount(a.Config), d.tenant).Scan(&ok); e != nil || !ok {
			continue // sin presupuesto no se encola nada
		}
		for _, t := range p.Topics {
			var used bool
			if e = a.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE tenant_id=$1 AND idempotency_key=$2)`, d.tenant, topicKey(d.site, t.Title)).Scan(&used); e != nil {
				return e
			}
			if used {
				continue
			}
			tx, e := a.DB.BeginTx(ctx, nil)
			if e != nil {
				return e
			}
			if e = enqueueTopic(ctx, tx, d.tenant, d.site, p, t); e != nil {
				tx.Rollback()
				return e
			}
			if e = tx.Commit(); e != nil {
				return e
			}
			break
		}
	}
	return nil
}

// ───────────── HTTP ─────────────

func (a *App) analyzeSite(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	if !a.Limiter.Allow("analyze:"+t, 6, time.Hour) {
		fail(w, 429, "Demasiados sitios en poco tiempo. Intenta más tarde.")
		return
	}
	var x struct {
		URL string `json:"url"`
	}
	if !decode(w, r, &x) {
		return
	}
	u, e := normalizeURL(x.URL)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	var avail bool
	if e = a.DB.QueryRowContext(r.Context(), `SELECT budget_microusd-spent_microusd-reserved_microusd>=$1 FROM tenants WHERE id=$2`, reserveAmount(a.Config), t).Scan(&avail); e != nil {
		dbError(w, e)
		return
	}
	if !avail {
		fail(w, 402, "Presupuesto agotado")
		return
	}
	domain := strings.TrimPrefix(u.Host, "www.")
	tx, e := a.DB.BeginTx(r.Context(), nil)
	if e != nil {
		dbError(w, e)
		return
	}
	defer tx.Rollback()
	var site string
	e = tx.QueryRowContext(r.Context(), `INSERT INTO sites(tenant_id,name,domain,canonical_url,autopilot) VALUES($1,$2,$3,$4,true) RETURNING id`, t, domain, domain, u.String()).Scan(&site)
	if e != nil {
		var pg *pgconn.PgError
		if errors.As(e, &pg) && pg.Code == "23505" {
			fail(w, 409, "Ese sitio ya está registrado")
		} else {
			dbError(w, e)
		}
		return
	}
	var job string
	e = tx.QueryRowContext(r.Context(), `INSERT INTO jobs(tenant_id,site_id,idempotency_key,topic,business,audience,kind) VALUES($1,$2,$3,$4,'','','research') RETURNING id`, t, site, "research-"+site, u.String()).Scan(&job)
	if e != nil || tx.Commit() != nil {
		dbError(w, errors.Join(e, errors.New("no se pudo crear la investigación")))
		return
	}
	jsonResponse(w, 200, map[string]string{"site_id": site, "job_id": job})
}

// research devuelve el estado de la investigación del sitio y lo que la IA dedujo.
func (a *App) siteResearch(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var blob []byte
	var auto, autoPublish bool
	var perWeek int
	e := a.DB.QueryRowContext(r.Context(), `SELECT research,autopilot,posts_per_week,auto_publish FROM sites WHERE id=$1 AND tenant_id=$2`, site, t).Scan(&blob, &auto, &perWeek, &autoPublish)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	var status string
	var msg sql.NullString
	e = a.DB.QueryRowContext(r.Context(), `SELECT status,error_message FROM jobs WHERE site_id=$1 AND tenant_id=$2 AND kind='research' ORDER BY created_at DESC LIMIT 1`, site, t).Scan(&status, &msg)
	if errors.Is(e, sql.ErrNoRows) {
		status = "none"
	} else if e != nil {
		dbError(w, e)
		return
	}
	out := map[string]any{"status": status, "autopilot": auto, "auto_publish": autoPublish, "posts_per_week": perWeek, "error": msg.String}
	if blob != nil {
		out["research"] = json.RawMessage(blob)
	}
	// avance de los artículos que arrancó el autopiloto
	var queued, done int
	_ = a.DB.QueryRowContext(r.Context(), `SELECT count(*) FILTER (WHERE status IN ('queued','running')),count(*) FILTER (WHERE status='done') FROM jobs WHERE site_id=$1 AND tenant_id=$2 AND kind='article'`, site, t).Scan(&queued, &done)
	out["articles_pending"], out["articles_done"] = queued, done
	jsonResponse(w, 200, out)
}

func (a *App) setAutopilot(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var x struct {
		Enabled      bool `json:"enabled"`
		PostsPerWeek int  `json:"posts_per_week"`
	}
	if !decode(w, r, &x) {
		return
	}
	if x.PostsPerWeek < 1 || x.PostsPerWeek > 3 { // más que esto en un sitio normal parece contenido masivo
		x.PostsPerWeek = 2
	}
	res, e := a.DB.ExecContext(r.Context(), `UPDATE sites SET autopilot=$1,posts_per_week=$2 WHERE id=$3 AND tenant_id=$4`, x.Enabled, x.PostsPerWeek, site, t)
	if e != nil {
		dbError(w, e)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	jsonResponse(w, 200, map[string]any{"autopilot": x.Enabled, "posts_per_week": x.PostsPerWeek})
}
