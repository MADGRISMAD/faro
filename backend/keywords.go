package main

// Datos reales de palabras clave (volumen mensual, CPC, competencia y dificultad) con DataForSEO.
// Sin credenciales (DATAFORSEO_LOGIN / DATAFORSEO_PASSWORD) el panel lo dice y NO inventa números.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type KwMetric struct {
	Volume           *int     `json:"volume"`
	Competition      *string  `json:"competition"`
	CompetitionIndex *int     `json:"competition_index"`
	CPC              *float64 `json:"cpc"`
	Difficulty       *int     `json:"difficulty"`
}

func dfsEnabled() bool {
	return env("DATAFORSEO_LOGIN", "") != "" && env("DATAFORSEO_PASSWORD", "") != ""
}
func dfsBase() string {
	return strings.TrimRight(env("DATAFORSEO_BASE", "https://api.dataforseo.com"), "/")
}

// México (2484) y español: los ids son los de Google Ads; se pueden cambiar sin tocar código.
func dfsLocation() int {
	var n int
	fmt.Sscanf(env("DATAFORSEO_LOCATION_CODE", "2484"), "%d", &n)
	if n == 0 {
		n = 2484
	}
	return n
}
func dfsLanguage() string { return env("DATAFORSEO_LANGUAGE_CODE", "es") }

// normKeyword: DataForSEO usa minúsculas, máximo 80 caracteres y 10 palabras por frase.
func normKeyword(k string) string {
	k = strings.ToLower(strings.Join(strings.Fields(k), " "))
	if k == "" || len(k) > 80 || len(strings.Fields(k)) > 10 {
		return ""
	}
	return k
}

// dfsPost hace una tarea y devuelve tasks[0].result tal cual. Revisa el estado general y el de la tarea (20000 = Ok).
func dfsPost(ctx context.Context, path string, task map[string]any) (json.RawMessage, error) {
	body, _ := json.Marshal([]any{task})
	req, e := http.NewRequestWithContext(ctx, "POST", dfsBase()+path, bytes.NewReader(body))
	if e != nil {
		return nil, e
	}
	req.SetBasicAuth(env("DATAFORSEO_LOGIN", ""), env("DATAFORSEO_PASSWORD", ""))
	req.Header.Set("Content-Type", "application/json")
	res, e := (&http.Client{Timeout: 25 * time.Second}).Do(req)
	if e != nil {
		return nil, errors.New("no se pudo contactar a DataForSEO")
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("DataForSEO respondió con estado %d", res.StatusCode)
	}
	var w struct {
		StatusCode int `json:"status_code"`
		Tasks      []struct {
			StatusCode    int             `json:"status_code"`
			StatusMessage string          `json:"status_message"`
			Result        json.RawMessage `json:"result"`
		} `json:"tasks"`
	}
	if json.Unmarshal(data, &w) != nil || w.StatusCode != 20000 || len(w.Tasks) == 0 {
		return nil, errors.New("respuesta inválida de DataForSEO")
	}
	if w.Tasks[0].StatusCode != 20000 {
		return nil, fmt.Errorf("DataForSEO: %s (%d)", w.Tasks[0].StatusMessage, w.Tasks[0].StatusCode)
	}
	return w.Tasks[0].Result, nil
}

// fetchMetrics pide volumen y dificultad de hasta 1000 palabras (2 llamadas, el precio es por llamada y no por palabra).
func fetchMetrics(ctx context.Context, kws []string) (map[string]KwMetric, error) {
	out := map[string]KwMetric{}
	if len(kws) == 0 {
		return out, nil
	}
	loc, lang := dfsLocation(), dfsLanguage()
	raw, e := dfsPost(ctx, "/v3/keywords_data/google_ads/search_volume/live", map[string]any{"keywords": kws, "location_code": loc, "language_code": lang})
	if e != nil {
		return nil, e
	}
	var vol []struct {
		Keyword          string   `json:"keyword"`
		SearchVolume     *int     `json:"search_volume"`
		Competition      *string  `json:"competition"`
		CompetitionIndex *int     `json:"competition_index"`
		CPC              *float64 `json:"cpc"`
	}
	if e = json.Unmarshal(raw, &vol); e != nil {
		return nil, errors.New("respuesta de volumen ilegible")
	}
	for _, v := range vol {
		out[strings.ToLower(v.Keyword)] = KwMetric{Volume: v.SearchVolume, Competition: v.Competition, CompetitionIndex: v.CompetitionIndex, CPC: v.CPC}
	}
	// La dificultad es un extra: si falla, el volumen igual se guarda.
	if raw, e = dfsPost(ctx, "/v3/dataforseo_labs/google/bulk_keyword_difficulty/live", map[string]any{"keywords": kws, "location_code": loc, "language_code": lang}); e == nil {
		var dif []struct {
			Items []struct {
				Keyword string `json:"keyword"`
				KD      *int   `json:"keyword_difficulty"`
			} `json:"items"`
		}
		if json.Unmarshal(raw, &dif) == nil && len(dif) > 0 {
			for _, it := range dif[0].Items {
				m := out[strings.ToLower(it.Keyword)]
				m.Difficulty = it.KD
				out[strings.ToLower(it.Keyword)] = m
			}
		}
	}
	return out, nil
}

// enrichKeywords completa la caché con las palabras de los temas del sitio que falten o tengan más de 30 días.
func (a *App) enrichKeywords(ctx context.Context, siteID string) (int, error) {
	if !dfsEnabled() {
		return 0, errors.New("datos de palabras clave no configurados")
	}
	var blob []byte
	if e := a.DB.QueryRowContext(ctx, `SELECT research FROM sites WHERE id=$1`, siteID).Scan(&blob); e != nil || blob == nil {
		return 0, errors.New("el sitio aún no tiene temas")
	}
	var p Profile
	if json.Unmarshal(blob, &p) != nil {
		return 0, errors.New("perfil ilegible")
	}
	loc, lang := dfsLocation(), dfsLanguage()
	var need []string
	seen := map[string]bool{}
	for _, t := range p.Topics {
		k := normKeyword(t.Keyword)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		var fresh bool
		_ = a.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM keyword_cache WHERE keyword=$1 AND location=$2 AND language=$3 AND fetched_at>now()-interval '30 days')`, k, loc, lang).Scan(&fresh)
		if !fresh {
			need = append(need, k)
		}
	}
	if len(need) == 0 {
		return 0, nil
	}
	m, e := fetchMetrics(ctx, need)
	if e != nil {
		return 0, e
	}
	for _, k := range need {
		x := m[k]
		if _, e = a.DB.ExecContext(ctx, `INSERT INTO keyword_cache(keyword,location,language,volume,competition,competition_index,cpc,difficulty) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(keyword,location,language) DO UPDATE SET volume=EXCLUDED.volume,competition=EXCLUDED.competition,competition_index=EXCLUDED.competition_index,cpc=EXCLUDED.cpc,difficulty=EXCLUDED.difficulty,fetched_at=now()`, k, loc, lang, x.Volume, x.Competition, x.CompetitionIndex, x.CPC, x.Difficulty); e != nil {
			return 0, e
		}
	}
	return len(need), nil
}

func (a *App) metricFor(ctx context.Context, keyword string) (KwMetric, bool) {
	var m KwMetric
	var cpc sql.NullFloat64
	e := a.DB.QueryRowContext(ctx, `SELECT volume,competition,competition_index,cpc,difficulty FROM keyword_cache WHERE keyword=$1 AND location=$2 AND language=$3`, normKeyword(keyword), dfsLocation(), dfsLanguage()).Scan(&m.Volume, &m.Competition, &m.CompetitionIndex, &cpc, &m.Difficulty)
	if cpc.Valid {
		m.CPC = &cpc.Float64
	}
	return m, e == nil
}

// rankTopics ordena por oportunidad real (mucho volumen y poca dificultad). Sin datos, deja el orden de la IA.
func (a *App) rankTopics(ctx context.Context, topics []Topic) []Topic {
	score := make([]float64, len(topics))
	any := false
	for i, t := range topics {
		if m, ok := a.metricFor(ctx, t.Keyword); ok && m.Volume != nil {
			d := 50.0
			if m.Difficulty != nil {
				d = float64(*m.Difficulty)
			}
			score[i] = float64(*m.Volume) / (d + 10)
			any = true
		}
	}
	if !any {
		return topics
	}
	idx := make([]int, len(topics))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(x, y int) bool { return score[idx[x]] > score[idx[y]] })
	out := make([]Topic, len(topics))
	for i, j := range idx {
		out[i] = topics[j]
	}
	return out
}

func (a *App) siteKeywords(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, false)
	if !ok {
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var blob []byte
	e := a.DB.QueryRowContext(r.Context(), `SELECT research FROM sites WHERE id=$1 AND tenant_id=$2`, site, t).Scan(&blob)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	var p Profile
	if blob != nil {
		_ = json.Unmarshal(blob, &p)
	}
	rows := []map[string]any{}
	for _, tp := range a.rankTopics(r.Context(), p.Topics) {
		var used bool
		_ = a.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM jobs WHERE tenant_id=$1 AND idempotency_key=$2)`, t, topicKey(site, tp.Title)).Scan(&used)
		row := map[string]any{"title": tp.Title, "keyword": tp.Keyword, "intent": tp.Intent, "why": tp.Why, "written": used}
		if m, ok := a.metricFor(r.Context(), tp.Keyword); ok {
			row["metrics"] = m
		}
		rows = append(rows, row)
	}
	jsonResponse(w, 200, map[string]any{"configured": dfsEnabled(), "location": dfsLocation(), "language": dfsLanguage(), "keywords": rows})
}

func (a *App) refreshKeywords(w http.ResponseWriter, r *http.Request) {
	_, t, ok := a.auth(w, r, true)
	if !ok {
		return
	}
	if !a.Limiter.Allow("kw:"+t, 6, time.Hour) {
		fail(w, 429, "Demasiadas actualizaciones. Intenta en un rato.")
		return
	}
	site, ok := requireUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var one int
	if e := a.DB.QueryRowContext(r.Context(), `SELECT 1 FROM sites WHERE id=$1 AND tenant_id=$2`, site, t).Scan(&one); e != nil {
		fail(w, 404, "Sitio no encontrado")
		return
	}
	if !dfsEnabled() {
		fail(w, 503, "Los datos de palabras clave todavía no están configurados en este servidor")
		return
	}
	n, e := a.enrichKeywords(r.Context(), site)
	if e != nil {
		fail(w, 502, e.Error())
		return
	}
	jsonResponse(w, 200, map[string]any{"updated": n})
}
