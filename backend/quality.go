package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Reglas de escritura de los artículos. Siguen las guías de contenido útil de Google (quién, cómo y por qué),
// E-E-A-T y lo que vuelve citable un pasaje: respuesta directa al inicio, definiciones claras, subtítulos
// descriptivos, párrafos cortos y nada de datos inventados.
func writerPrompt(p Prompt) string {
	var b strings.Builder
	b.WriteString("Escribe en español de México un artículo de blog original y útil para las personas, no para buscadores.\n\n")
	fmt.Fprintf(&b, "Tema: %s\nPalabra clave principal: %s\nNegocio que publica: %s\nQué es y qué ofrece el negocio (única fuente de datos sobre él): %s\nPúblico: %s\nTono: %s\n", p.Topic, orDash(p.Keyword), orDash(p.Name), p.Brand, p.Audience, p.Tone)
	if p.Location != "" {
		fmt.Fprintf(&b, "Ubicación del negocio: %s (menciónala solo donde ayude al lector, sin repetirla).\n", p.Location)
	}
	b.WriteString(`
Estructura:
- title: de 40 a 60 caracteres, con la palabra clave al principio, sin el nombre del negocio y sin signos de exclamación.
- meta_description: de 120 a 155 caracteres. Resume lo que el lector va a resolver y termina con una invitación pensada para este tema. No repitas el título.
- summary: una o dos frases.
- intro: responde la pregunta del tema en las dos primeras frases (si el tema es «qué es X», empieza con «X es…»). Usa la palabra clave en las primeras 100 palabras, de forma natural.
- sections: de 4 a 7 apartados. Cada heading describe lo que contiene (nada de títulos ingeniosos); al menos dos están escritos como la pregunta que haría un cliente. El primer párrafo de cada apartado responde de forma directa y se entiende solo, sin depender del resto. Párrafos de 2 a 4 frases cortas. Usa bullets para pasos, requisitos o comparaciones. Usa level 3 solo para un subtítulo dentro del apartado anterior.
- Incluye un apartado sobre límites o cuándo NO conviene lo que se explica: un buen consejo también dice lo que no resuelve.
- El último apartado da un siguiente paso concreto relacionado con lo que ofrece el negocio, sin presión de venta.
- faq: de 3 a 5 preguntas que el artículo no haya respondido ya con el mismo título; respuestas de 40 a 80 palabras que se entiendan solas.
- Entre 900 y 1,300 palabras en total. Si el tema se resuelve en menos, no rellenes.

Datos y afirmaciones:
- No inventes cifras, porcentajes, precios, plazos, estudios, encuestas, leyes, casos, clientes ni testimonios. Si no tienes el dato exacto en la descripción del negocio, explícalo sin número («depende de…», «suele reducir…»).
- No atribuyas al negocio funciones, servicios, certificaciones ni resultados que no estén en su descripción.
- Nada de garantías ni absolutos: no uses «garantiza», «100 %», «sin riesgo», «el mejor», «absoluto», «siempre funciona».
- No incluyas HTML, Markdown ni enlaces.

Ortografía:
- Español con ortografía completa: acentos, ñ, ¿ y ¡ en todo el texto, también en title, meta_description y headings.
- El tema y la palabra clave pueden venir sin acentos porque así los escribe la gente al buscar: en el artículo escríbelos correctamente acentuados («médicos», «México»).

Estilo:
- Concreto y práctico: ejemplos de la vida diaria del público, pasos que se puedan seguir.
- Evita frases de relleno típicas de textos generados: «en el mundo actual», «en la era digital», «hoy más que nunca», «es crucial», «sin lugar a dudas», «llevar tu negocio al siguiente nivel», «solución integral», «potenciar», «en conclusión», «en resumen», «descubre cómo», «sumérgete».
`)
	if p.Sensitive {
		b.WriteString("- El tema es de salud, legal o finanzas: explica con prudencia, aclara que no sustituye la orientación de un profesional y no des indicaciones personales.\n")
	}
	b.WriteString("\nDevuelve JSON con title, meta_description, summary y content (intro, sections con heading/level/paragraphs/bullets y faq con question/answer).")
	return b.String()
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

var (
	percentRe = regexp.MustCompile(`\d+(?:[.,]\d+)?\s?%`)
	timesRe   = regexp.MustCompile(`(?i)\b\d+(?:[.,]\d+)?\s+veces\b`)
	moneyRe   = regexp.MustCompile(`\$\s?\d[\d.,]*`)
	promiseRe = regexp.MustCompile(`(?i)\b(garantiz\p{L}*|sin (?:ningún )?riesgo|resultados inmediatos|absolut\p{L}*|número uno|el mejor del mercado|la mejor del mercado)\b`)
	accentRe  = regexp.MustCompile(`[áéíóúüñÁÉÍÓÚÜÑ¿¡]`)
	studyRe   = regexp.MustCompile(`(?i)\b(según (?:un |los |diversos |varios )?estudios?|estudios (?:demuestran|indican|revelan)|investigaciones (?:demuestran|indican)|una encuesta de)\b`)
)

// review busca lo que un artículo no debería publicar sin que alguien lo revise: cifras que no vienen de la
// descripción del negocio, promesas absolutas y estudios sin citar. Si hay algo, el artículo queda en borrador.
func review(g Generated, facts string) []string {
	text := articleText(g)
	notes := []string{}
	seen := map[string]bool{}
	add := func(n string) {
		if !seen[n] && len(notes) < 8 {
			seen[n] = true
			notes = append(notes, n)
		}
	}
	for _, re := range []*regexp.Regexp{percentRe, timesRe, moneyRe} {
		for _, m := range re.FindAllString(text, -1) {
			if !strings.Contains(facts, strings.TrimSpace(m)) {
				add("Cifra sin fuente: «" + strings.TrimSpace(m) + "». Confírmala o quítala.")
			}
		}
	}
	for _, m := range promiseRe.FindAllString(text, -1) {
		add("Promesa absoluta: «" + m + "». Suavízala si no puedes respaldarla.")
	}
	for _, m := range studyRe.FindAllString(text, -1) {
		add("Menciona «" + m + "» sin decir cuál. Cita la fuente o quítalo.")
	}
	// un texto largo en español sin un solo acento salió mal escrito (pasa cuando el tema llega sin acentos)
	if len(strings.Fields(text)) > 150 && !accentRe.MatchString(text) {
		add("El texto no tiene acentos ni signos de apertura. Corrige la ortografía o vuelve a generarlo.")
	}
	return notes
}

func articleText(g Generated) string {
	parts := []string{g.Title, g.MetaDescription, g.Summary, g.Content.Intro}
	for _, s := range g.Content.Sections {
		parts = append(parts, s.Heading)
		parts = append(parts, s.Paragraphs...)
		parts = append(parts, s.Bullets...)
	}
	for _, f := range g.Content.FAQ {
		parts = append(parts, f.Question, f.Answer)
	}
	return strings.Join(parts, "\n")
}

// Palabras que no aportan a la dirección del artículo: «que-es-un-sistema-a-la-medida…» → «que-es-sistema-medida…».
var slugStop = map[string]bool{"a": true, "al": true, "de": true, "del": true, "el": true, "en": true, "la": true, "las": true, "lo": true, "los": true, "un": true, "una": true, "unos": true, "unas": true, "y": true, "o": true, "u": true, "e": true, "por": true, "para": true, "con": true, "su": true, "sus": true, "tu": true, "tus": true, "se": true, "le": true, "les": true, "es": false, "que": false}

// shortSlug: corta y legible (hasta 7 palabras con significado, 60 caracteres).
func shortSlug(title string) string {
	words := []string{}
	for _, w := range strings.Split(slug(title), "-") {
		if w == "" || (slugStop[w] && len(words) > 0) {
			continue
		}
		if len(strings.Join(append(words, w), "-")) > 60 || len(words) == 7 {
			break
		}
		words = append(words, w)
	}
	if len(words) == 0 {
		return "articulo"
	}
	return strings.Join(words, "-")
}

// trimMeta deja la descripción en 160 caracteres como máximo, cortando en un espacio.
func trimMeta(s string) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= 160 {
		return s
	}
	r := []rune(s)[:160]
	if i := strings.LastIndex(string(r), " "); i > 100 {
		return strings.TrimRight(string(r)[:i], " ,;:") + "…"
	}
	return string(r)
}

// invisible: caracteres de ancho cero, marcas de dirección y etiquetas Unicode (se usan para marcar o esconder texto).
func invisible(r rune) bool {
	return (r >= 0x200B && r <= 0x200F) || (r >= 0x202A && r <= 0x202E) || (r >= 0x2060 && r <= 0x2064) || r == 0xFEFF || (r >= 0xE0000 && r <= 0xE007F)
}
