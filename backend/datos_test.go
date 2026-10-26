package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

const testKey = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

func siteWithTopics(t *testing.T, app *App, srv *httptest.Server) (cookie, csrf, tenant, siteID, domain string) {
	t.Helper()
	cookie, csrf, tenant = registerTest(t, srv)
	domain = "datos-" + token()[:8] + ".test"
	_, res, _ := request(t, srv, "POST", "/sites/analyze", map[string]string{"url": domain}, cookie, csrf, "")
	siteID = res["site_id"].(string)
	app.DB.Exec(`UPDATE jobs SET status='failed' WHERE site_id=$1`, siteID)
	prof, _ := json.Marshal(Profile{Name: "X", Business: "Negocio", Audience: "Público", Topics: []Topic{
		{Title: "Tema difícil", Keyword: "Software Médico", Intent: "informational"},
		{Title: "Tema oportunidad", Keyword: "agenda médica", Intent: "purchase"},
		{Title: "Tema sin datos", Keyword: "otra cosa rara", Intent: "local"},
	}})
	app.DB.Exec(`UPDATE sites SET research=$1,brand_confirmed_at=now() WHERE id=$2`, prof, siteID)
	return
}

func TestPalabrasClaveConDatosReales(t *testing.T) {
	var volCalls, kdCalls atomic.Int32
	dfs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "login-prueba" || p != "clave-prueba" {
			http.Error(w, "no autorizado", 401)
			return
		}
		var tasks []map[string]any
		json.NewDecoder(r.Body).Decode(&tasks)
		task := tasks[0]
		if task["location_code"].(float64) != 2484 || task["language_code"] != "es" {
			t.Errorf("ubicación/idioma: %v %v", task["location_code"], task["language_code"])
		}
		for _, k := range task["keywords"].([]any) {
			if k.(string) != strings.ToLower(k.(string)) {
				t.Errorf("las palabras deben ir en minúsculas: %v", k)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v3/keywords_data/google_ads/search_volume/live":
			volCalls.Add(1)
			w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[
				{"keyword":"software médico","search_volume":3200,"competition":"HIGH","competition_index":88,"cpc":4.5},
				{"keyword":"agenda médica","search_volume":6600,"competition":"LOW","competition_index":20,"cpc":1.2},
				{"keyword":"otra cosa rara","search_volume":null,"competition":null,"competition_index":null,"cpc":null}]}]}`))
		case "/v3/dataforseo_labs/google/bulk_keyword_difficulty/live":
			kdCalls.Add(1)
			w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"items":[
				{"keyword":"software médico","keyword_difficulty":71},{"keyword":"agenda médica","keyword_difficulty":24}]}]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer dfs.Close()
	app := testApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()
	cookie, csrf, _, siteID, _ := siteWithTopics(t, app, srv)
	app.DB.Exec(`DELETE FROM keyword_cache`)

	// sin credenciales: no se inventa nada
	if code, _, _ := request(t, srv, "POST", "/sites/"+siteID+"/keywords/refresh", map[string]any{}, cookie, csrf, ""); code != 503 {
		t.Fatalf("sin credenciales debía ser 503: %d", code)
	}
	_, kw, _ := request(t, srv, "GET", "/sites/"+siteID+"/keywords", nil, cookie, "", "")
	if kw["configured"] != false {
		t.Fatalf("configured: %#v", kw)
	}
	for _, row := range kw["keywords"].([]any) {
		if row.(map[string]any)["metrics"] != nil {
			t.Fatal("mostró métricas sin tener ninguna fuente")
		}
	}

	t.Setenv("DATAFORSEO_LOGIN", "login-prueba")
	t.Setenv("DATAFORSEO_PASSWORD", "clave-prueba")
	t.Setenv("DATAFORSEO_BASE", dfs.URL)
	if code, res, _ := request(t, srv, "POST", "/sites/"+siteID+"/keywords/refresh", map[string]any{}, cookie, csrf, ""); code != 200 || res["updated"].(float64) != 3 {
		t.Fatalf("actualizar: %d %#v", code, res)
	}
	if code, res, _ := request(t, srv, "POST", "/sites/"+siteID+"/keywords/refresh", map[string]any{}, cookie, csrf, ""); code != 200 || res["updated"].(float64) != 0 {
		t.Fatalf("la caché fresca no debe volver a pedir: %d %#v", code, res)
	}
	if volCalls.Load() != 1 || kdCalls.Load() != 1 {
		t.Fatalf("llamadas: volumen=%d dificultad=%d (una de cada una por lote, no por palabra)", volCalls.Load(), kdCalls.Load())
	}
	_, kw, _ = request(t, srv, "GET", "/sites/"+siteID+"/keywords", nil, cookie, "", "")
	rows := kw["keywords"].([]any)
	first := rows[0].(map[string]any)
	m := first["metrics"].(map[string]any)
	if first["title"] != "Tema oportunidad" || m["volume"].(float64) != 6600 || m["difficulty"].(float64) != 24 {
		t.Fatalf("debía ir primero el de más oportunidad (volumen alto, dificultad baja): %#v", rows)
	}
	last := rows[2].(map[string]any)
	if last["title"] != "Tema sin datos" || last["metrics"].(map[string]any)["volume"] != nil {
		t.Fatalf("sin dato real debe quedar vacío, no inventado: %#v", last)
	}
	// otro cliente no ve este sitio
	other, _, _ := registerTest(t, srv)
	if code, _, _ := request(t, srv, "GET", "/sites/"+siteID+"/keywords", nil, other, "", ""); code != 404 {
		t.Fatalf("palabras clave ajenas: %d", code)
	}
	// error de la tarea: se informa, no se guarda nada
	app.DB.Exec(`DELETE FROM keyword_cache`)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":40501,"status_message":"Invalid Field: 'location_code'."}]}`))
	}))
	defer bad.Close()
	t.Setenv("DATAFORSEO_BASE", bad.URL)
	if code, res, _ := request(t, srv, "POST", "/sites/"+siteID+"/keywords/refresh", map[string]any{}, cookie, csrf, ""); code != 502 || !strings.Contains(res["error"].(string), "40501") {
		t.Fatalf("error de tarea: %d %#v", code, res)
	}
	var n int
	app.DB.QueryRow(`SELECT count(*) FROM keyword_cache`).Scan(&n)
	if n != 0 {
		t.Fatalf("guardó datos de una tarea fallida: %d", n)
	}
}

func TestGoogleSearchConsoleReal(t *testing.T) {
	var apiCalls, revoked atomic.Int32
	var domain string
	g := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/token":
			r.ParseForm()
			switch r.Form.Get("grant_type") {
			case "authorization_code":
				if r.Form.Get("code") != "codigo-ok" || r.Form.Get("client_id") != "cid" || r.Form.Get("client_secret") != "csecret" || r.Form.Get("redirect_uri") == "" {
					http.Error(w, "bad", 400)
					return
				}
				w.Write([]byte(`{"access_token":"acceso-1","refresh_token":"refresh-secreto-xyz"}`))
			case "refresh_token":
				if r.Form.Get("refresh_token") != "refresh-secreto-xyz" {
					http.Error(w, "bad", 400)
					return
				}
				w.Write([]byte(`{"access_token":"acceso-2"}`))
			}
		case r.URL.Path == "/revoke":
			r.ParseForm()
			if r.Form.Get("token") == "refresh-secreto-xyz" {
				revoked.Add(1)
			}
		case r.URL.Path == "/webmasters/v3/sites":
			if r.Header.Get("Authorization") != "Bearer acceso-1" {
				http.Error(w, "no", 401)
				return
			}
			w.Write([]byte(`{"siteEntry":[{"siteUrl":"https://otro.com/","permissionLevel":"siteOwner"},{"siteUrl":"sc-domain:` + domain + `","permissionLevel":"siteOwner"}]}`))
		case strings.HasSuffix(r.URL.EscapedPath(), "/searchAnalytics/query"):
			apiCalls.Add(1)
			if r.Header.Get("Authorization") != "Bearer acceso-2" || !strings.Contains(r.URL.EscapedPath(), "sites/sc-domain%3A") {
				http.Error(w, "no", 401)
				return
			}
			var b map[string]any
			json.NewDecoder(r.Body).Decode(&b)
			dims, _ := b["dimensions"].([]any)
			switch {
			case len(dims) == 0 && b["startDate"].(string) > "2000": // totales: el periodo actual y el anterior
				c := 120.0
				if b["endDate"].(string) < "2026-10-01" {
					c = 60
				}
				w.Write([]byte(`{"rows":[{"clicks":` + jsonNum(c) + `,"impressions":4000,"ctr":0.03,"position":8.4}]}`))
			case dims[0] == "date":
				w.Write([]byte(`{"rows":[{"keys":["2026-10-20"],"clicks":4,"impressions":150},{"keys":["2026-10-21"],"clicks":6,"impressions":210}]}`))
			case dims[0] == "query":
				w.Write([]byte(`{"rows":[{"keys":["software a la medida"],"clicks":40,"impressions":900,"ctr":0.044,"position":6.2}]}`))
			case dims[0] == "page":
				w.Write([]byte(`{"rows":[{"keys":["https://x.test/blog/a"],"clicks":30,"impressions":700,"ctr":0.042,"position":7.1}]}`))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer g.Close()
	app := testApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()
	cookie, csrf, _, siteID, dom := siteWithTopics(t, app, srv)
	domain = dom
	otherCookie, otherCSRF, _ := registerTest(t, srv)

	// apagada sin configuración
	if code, _, _ := request(t, srv, "POST", "/sites/"+siteID+"/google/start", map[string]any{}, cookie, csrf, ""); code != 503 {
		t.Fatalf("sin configurar debía ser 503: %d", code)
	}
	if _, a, _ := request(t, srv, "GET", "/sites/"+siteID+"/analytics", nil, cookie, "", ""); a["connected"] != false || a["configured"] != false {
		t.Fatalf("estado sin configurar: %#v", a)
	}
	t.Setenv("GOOGLE_CLIENT_ID", "cid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
	t.Setenv("TOKEN_ENCRYPTION_KEY", testKey)
	t.Setenv("GOOGLE_REDIRECT_URI", "https://faro.test/api/google/callback")
	t.Setenv("GOOGLE_AUTH_URL", g.URL+"/auth")
	t.Setenv("GOOGLE_TOKEN_URL", g.URL+"/token")
	t.Setenv("GOOGLE_REVOKE_URL", g.URL+"/revoke")
	t.Setenv("GOOGLE_API_BASE", g.URL+"/webmasters/v3")

	if code, _, _ := request(t, srv, "POST", "/sites/"+siteID+"/google/start", map[string]any{}, otherCookie, otherCSRF, ""); code != 404 {
		t.Fatalf("conectar el sitio de otro: %d", code)
	}
	code, st, _ := request(t, srv, "POST", "/sites/"+siteID+"/google/start", map[string]any{}, cookie, csrf, "")
	if code != 200 {
		t.Fatalf("start: %d %#v", code, st)
	}
	u, _ := url.Parse(st["url"].(string))
	q := u.Query()
	if q.Get("scope") != gScope || q.Get("access_type") != "offline" || q.Get("prompt") != "consent" || q.Get("client_id") != "cid" || q.Get("redirect_uri") != "https://faro.test/api/google/callback" || q.Get("response_type") != "code" {
		t.Fatalf("parámetros OAuth: %v", q)
	}
	state := q.Get("state")
	call := func(state, code string) string {
		req, _ := http.NewRequest("GET", srv.URL+"/google/callback?state="+url.QueryEscape(state)+"&code="+code, nil)
		c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		res, e := c.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		return res.Header.Get("Location")
	}
	if loc := call(state+"x", "codigo-ok"); !strings.HasSuffix(loc, "google=error") {
		t.Fatalf("state alterado: %s", loc)
	}
	var n int
	app.DB.QueryRow(`SELECT count(*) FROM site_google WHERE site_id=$1`, siteID).Scan(&n)
	if n != 0 {
		t.Fatal("conectó con un state falso")
	}
	if loc := call(state, "codigo-ok"); loc != "/dashboard/analytics?google=ok" {
		t.Fatalf("callback: %s", loc)
	}
	var enc []byte
	var prop string
	app.DB.QueryRow(`SELECT refresh_enc,property FROM site_google WHERE site_id=$1`, siteID).Scan(&enc, &prop)
	if prop != "sc-domain:"+domain || strings.Contains(string(enc), "refresh-secreto-xyz") || strings.Contains(hex.EncodeToString(enc), hex.EncodeToString([]byte("refresh-secreto"))) {
		t.Fatalf("propiedad=%q; el token debe guardarse cifrado", prop)
	}
	if plain, e := unseal(enc); e != nil || string(plain) != "refresh-secreto-xyz" {
		t.Fatalf("descifrar: %v %s", e, plain)
	}

	_, an, _ := request(t, srv, "GET", "/sites/"+siteID+"/analytics?range=28d", nil, cookie, "", "")
	tot := an["totals"].(map[string]any)
	if an["connected"] != true || tot["clicks"].(float64) != 120 || an["previous"].(map[string]any)["clicks"].(float64) != 60 || tot["position"].(float64) != 8.4 {
		t.Fatalf("totales: %#v", an)
	}
	if len(an["series"].([]any)) != 2 || an["queries"].([]any)[0].(map[string]any)["query"] != "software a la medida" || an["pages"].([]any)[0].(map[string]any)["page"] != "https://x.test/blog/a" {
		t.Fatalf("series/consultas/páginas: %#v", an)
	}
	before := apiCalls.Load()
	request(t, srv, "GET", "/sites/"+siteID+"/analytics?range=28d", nil, cookie, "", "")
	if apiCalls.Load() != before {
		t.Fatal("la segunda lectura debía salir de la caché de 1 hora")
	}
	if code, _, _ := request(t, srv, "GET", "/sites/"+siteID+"/analytics", nil, otherCookie, "", ""); code != 404 {
		t.Fatalf("analytics ajeno: %d", code)
	}

	// desconectar: se revoca en Google y se borra el token
	if code, _, _ := request(t, srv, "DELETE", "/sites/"+siteID+"/google", nil, cookie, csrf, ""); code != 200 {
		t.Fatalf("desconectar: %d", code)
	}
	app.DB.QueryRow(`SELECT count(*) FROM site_google WHERE site_id=$1`, siteID).Scan(&n)
	if n != 0 || revoked.Load() != 1 {
		t.Fatalf("token borrado=%v revocado=%d", n == 0, revoked.Load())
	}
	// una cuenta sin el dominio verificado queda conectada pero sin propiedad
	state2 := signState(siteID, mustTenant(t, app, siteID))
	domain = "otro-dominio.com"
	if loc := call(state2, "codigo-ok"); !strings.HasSuffix(loc, "google=noproperty") {
		t.Fatalf("sin propiedad: %s", loc)
	}
	_, an, _ = request(t, srv, "GET", "/sites/"+siteID+"/analytics", nil, cookie, "", "")
	if an["property"] != "" || an["error"] == nil {
		t.Fatalf("sin propiedad: %#v", an)
	}
	_ = context.Background()
}

func jsonNum(f float64) string { b, _ := json.Marshal(f); return string(b) }
func mustTenant(t *testing.T, app *App, site string) string {
	var id string
	if e := app.DB.QueryRow(`SELECT tenant_id FROM sites WHERE id=$1`, site).Scan(&id); e != nil {
		t.Fatal(e)
	}
	return id
}
