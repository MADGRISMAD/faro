package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// La paleta de la marca sale de su propio sitio (theme-color y su CSS): las portadas usan solo esos colores,
// más blanco y un tono oscuro para los contornos, para que se vean de la misma casa que la página.

var (
	reThemeColor = regexp.MustCompile(`(?i)<meta[^>]+name=["']theme-color["'][^>]*content=["']([^"']+)["']|<meta[^>]+content=["']([^"']+)["'][^>]*name=["']theme-color["']`)
	reStyleTag   = regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`)
	reStyleAttr  = regexp.MustCompile(`(?i)style=["']([^"']+)["']`)
	reSheet      = regexp.MustCompile(`(?i)<link[^>]+rel=["']?stylesheet["']?[^>]*>`)
	reSheetHref  = regexp.MustCompile(`(?i)href=["']([^"']+)["']`)
	reHex        = regexp.MustCompile(`#([0-9a-fA-F]{8}|[0-9a-fA-F]{6}|[0-9a-fA-F]{3,4})\b`)
	reRGB        = regexp.MustCompile(`(?i)rgba?\(\s*(\d{1,3})[\s,]+(\d{1,3})[\s,]+(\d{1,3})`)
	reBrandVar   = regexp.MustCompile(`(?i)--[a-z0-9-]*(primary|accent|brand|main|principal|secondary|highlight)[a-z0-9-]*\s*:\s*([^;}\n]+)`)
)

type rgb struct{ r, g, b float64 }

func (c rgb) hex() string { return fmt.Sprintf("#%02x%02x%02x", int(c.r), int(c.g), int(c.b)) }

// lightness y saturación (HSL) en 0..1
func (c rgb) hsl() (s, l float64) {
	r, g, b := c.r/255, c.g/255, c.b/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, l
	}
	d := mx - mn
	if l > 0.5 {
		return d / (2 - mx - mn), l
	}
	return d / (mx + mn), l
}

func parseColor(s string) (rgb, bool) {
	s = strings.TrimSpace(s)
	if m := reHex.FindStringSubmatch(s); m != nil {
		h := m[1]
		if len(h) == 3 || len(h) == 4 {
			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
		}
		h = h[:6]
		v, e := strconv.ParseUint(h, 16, 32)
		if e != nil {
			return rgb{}, false
		}
		return rgb{float64(v >> 16 & 255), float64(v >> 8 & 255), float64(v & 255)}, true
	}
	if m := reRGB.FindStringSubmatch(s); m != nil {
		r, _ := strconv.Atoi(m[1])
		g, _ := strconv.Atoi(m[2])
		b, _ := strconv.Atoi(m[3])
		if r > 255 || g > 255 || b > 255 {
			return rgb{}, false
		}
		return rgb{float64(r), float64(g), float64(b)}, true
	}
	return rgb{}, false
}

// extractPalette devuelve hasta 4 colores de marca (sin blancos, negros ni grises), del más usado al menos usado.
// theme-color y las variables tipo --primary/--accent/--brand pesan más que un color suelto.
func extractPalette(html string, css []string) []string {
	score := map[string]float64{}
	colors := map[string]rgb{}
	add := func(s string, w float64) {
		for _, raw := range append(reHex.FindAllString(s, -1), reRGB.FindAllString(s, -1)...) {
			c, ok := parseColor(raw)
			if !ok {
				continue
			}
			sat, l := c.hsl()
			if l > 0.93 || l < 0.1 || sat < 0.2 {
				continue
			}
			k := c.hex()
			colors[k] = c
			score[k] += w
		}
	}
	for _, m := range reThemeColor.FindAllStringSubmatch(html, -1) {
		add(m[1]+" "+m[2], 8)
	}
	sheets := []string{}
	for _, m := range reStyleTag.FindAllStringSubmatch(html, -1) {
		sheets = append(sheets, m[1])
	}
	for _, m := range reStyleAttr.FindAllStringSubmatch(html, -1) {
		sheets = append(sheets, m[1])
	}
	sheets = append(sheets, css...)
	for _, s := range sheets {
		for _, m := range reBrandVar.FindAllStringSubmatch(s, -1) {
			add(m[2], 4)
		}
		add(s, 1)
	}
	keys := make([]string, 0, len(score))
	for k := range score {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if score[keys[i]] != score[keys[j]] {
			return score[keys[i]] > score[keys[j]]
		}
		return keys[i] < keys[j]
	})
	out := []string{}
	picked := []rgb{}
	for _, k := range keys {
		c := colors[k]
		near := false
		for _, p := range picked { // tonos casi iguales (hover, bordes) cuentan como uno
			if math.Abs(c.r-p.r)+math.Abs(c.g-p.g)+math.Abs(c.b-p.b) < 60 {
				near = true
				break
			}
		}
		if near {
			continue
		}
		picked = append(picked, c)
		out = append(out, k)
		if len(out) == 4 {
			break
		}
	}
	return out
}

// sitePalette: la paleta guardada del sitio o, la primera vez, la que se lee de su portada y sus hojas de estilo propias.
func (a *App) sitePalette(ctx context.Context, site string) []string {
	var saved []byte
	var home string
	if a.DB.QueryRowContext(ctx, `SELECT COALESCE(research->'palette','null'),canonical_url FROM sites WHERE id=$1`, site).Scan(&saved, &home) != nil {
		return nil
	}
	var p []string
	if json.Unmarshal(saved, &p) == nil && p != nil {
		return p
	}
	c := safeClient()
	html, final, _, e := fetchPage(ctx, c, home)
	if e != nil {
		return nil
	}
	css := []string{}
	for _, tag := range reSheet.FindAllString(html, 4) {
		m := reSheetHref.FindStringSubmatch(tag)
		if m == nil {
			continue
		}
		u, e := final.Parse(m[1])
		if e != nil || u.Host != final.Host { // solo las hojas del propio sitio, no fuentes ni librerías externas
			continue
		}
		if s, e := fetchCSS(ctx, c, u); e == nil {
			css = append(css, s)
		}
	}
	p = extractPalette(html, css)
	b, _ := json.Marshal(p)
	_, _ = a.DB.ExecContext(ctx, `UPDATE sites SET research=jsonb_set(research,'{palette}',$1::jsonb) WHERE id=$2 AND research IS NOT NULL`, string(b), site)
	return p
}

func fetchCSS(ctx context.Context, c *http.Client, u *url.URL) (string, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if e != nil {
		return "", e
	}
	req.Header.Set("User-Agent", "FaroAiBot/1.0 (+lectura de tu sitio para planear contenido)")
	req.Header.Set("Accept", "text/css")
	res, e := c.Do(req)
	if e != nil {
		return "", e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Content-Type"), "css") {
		return "", fmt.Errorf("hoja de estilo no disponible")
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, 800000))
	return string(b), e
}
