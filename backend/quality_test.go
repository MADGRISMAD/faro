package main

import (
	"strings"
	"testing"
	"time"
)

func TestReviewFlagsWhatCannotBeBacked(t *testing.T) {
	g := Generated{Title: "Qué es un sistema a la medida", Content: Content{
		Intro:    "Un sistema a la medida reduce hasta un 70% el tiempo de captura y te lo garantizamos.",
		Sections: []Section{{Heading: "Costos", Paragraphs: []string{"Según estudios, cuesta $15,000 y ahorra 3 veces más. El plan cuesta $499 al mes."}}},
	}}
	notes := strings.Join(review(g, "Caresia: planes desde $499 al mes."), "\n")
	for _, want := range []string{"«70%»", "«garantizamos»", "«$15,000»", "«3 veces»", "Según estudios"} {
		if !strings.Contains(notes, want) {
			t.Errorf("faltó %s en:\n%s", want, notes)
		}
	}
	if strings.Contains(notes, "$499") {
		t.Error("$499 está en los datos del negocio: no es una cifra sin fuente")
	}
	sinAcentos := Generated{Title: "Mejor software para consultorios medicos en Mexico", Content: Content{Intro: strings.Repeat("El software para consultorios medicos ayuda a la clinica con la informacion. ", 20)}}
	if n := strings.Join(review(sinAcentos, ""), " "); !strings.Contains(n, "no tiene acentos") {
		t.Errorf("texto sin acentos no marcado: %s", n)
	}
	if n := review(Generated{Title: "Cómo ordenar tu inventario", Content: Content{Intro: "Empieza por contar lo que tienes."}}, ""); len(n) != 0 {
		t.Errorf("texto limpio marcado: %v", n)
	}
}

func TestShortSlugAndMeta(t *testing.T) {
	if s := shortSlug("¿Qué es un sistema a la medida y cuándo le conviene a tu negocio?"); s != "que-es-sistema-medida-cuando-conviene-negocio" {
		t.Errorf("slug: %s", s)
	}
	if s := shortSlug("La guía"); s != "la-guia" {
		t.Errorf("la primera palabra se conserva aunque sea artículo: %s", s)
	}
	if s := shortSlug("???"); s != "articulo" {
		t.Errorf("vacío: %s", s)
	}
	long := strings.Repeat("palabra ", 40)
	if m := trimMeta(long); len([]rune(m)) > 161 || !strings.HasSuffix(m, "…") {
		t.Errorf("meta: %d %q", len([]rune(m)), m)
	}
	if m := trimMeta(" corta "); m != "corta" {
		t.Errorf("meta corta: %q", m)
	}
}

func TestCleanDropsInvisibleCharacters(t *testing.T) {
	if s := clean("ho​la‮ mundo\U000E0041", 100); s != "hola mundo" {
		t.Errorf("%q", s)
	}
}

func TestWriterPromptCarriesTheRules(t *testing.T) {
	p := writerPrompt(Prompt{Topic: "Expediente clínico", Brand: "Caresia: agenda y expediente.", Name: "Caresia", Location: "Tijuana", Sensitive: true})
	for _, want := range []string{"Expediente clínico", "Caresia: agenda y expediente.", "Tijuana", "No inventes cifras", "acentos, ñ, ¿ y ¡", "40 a 60 caracteres", "120 a 155", "cuándo NO conviene", "no sustituye la orientación de un profesional"} {
		if !strings.Contains(p, want) {
			t.Errorf("falta %q", want)
		}
	}
	if strings.Contains(writerPrompt(Prompt{Topic: "x"}), "no sustituye") {
		t.Error("el aviso de salud/legal solo va en sitios sensibles")
	}
}

func TestJSONLDLinksAuthorAndCrumbs(t *testing.T) {
	s := v1Site{Name: "Hyperion Software", Canonical: "https://www.hyperionmkt.com", Prefix: "/blog"}
	out := jsonLD(s, "Título", "desc", "mi-articulo", time.Now(), false, Content{})
	post := out[0].(map[string]any)
	org := post["author"].(map[string]any)
	if org["@id"] != "https://www.hyperionmkt.com/#organization" || post["publisher"].(map[string]any)["@id"] != org["@id"] || post["inLanguage"] != "es-MX" {
		t.Fatalf("autor/editor: %v", post)
	}
	if post["@id"] != "https://www.hyperionmkt.com/blog/mi-articulo#article" || out[1].(map[string]any)["@type"] != "BreadcrumbList" {
		t.Fatalf("id o migas: %v", out)
	}
	if h := relatedHTML([]map[string]string{{"title": "A <b>", "path": "/blog/a"}}); !strings.Contains(h, `<a href="/blog/a">A &lt;b&gt;</a>`) {
		t.Errorf("relacionados: %s", h)
	}
	if relatedHTML(nil) != "" {
		t.Error("sin relacionados no se agrega nada")
	}
}
