package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Prompt struct {
	Topic, Business, Audience, Keyword, Brand, Tone string
	Name, Location                                  string // nombre del sitio y ubicación del negocio (del Brand DNA)
	Sensitive                                       bool   // salud, legal o finanzas
}
type Generated struct {
	Title                     string  `json:"title"`
	MetaDescription           string  `json:"meta_description"`
	Summary                   string  `json:"summary"`
	Content                   Content `json:"content"`
	InputTokens, OutputTokens int     `json:"-"`
}
type Content struct {
	Intro    string    `json:"intro"`
	Sections []Section `json:"sections"`
	FAQ      []FAQ     `json:"faq"`
}
type Section struct {
	Heading    string   `json:"heading"`
	Level      int      `json:"level,omitempty"` // 3 = subtítulo dentro del apartado anterior; vacío = apartado normal (h2)
	Paragraphs []string `json:"paragraphs"`
	Bullets    []string `json:"bullets"`
}
type FAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}
type ProviderResult struct {
	Article                   Generated
	InputTokens, OutputTokens int
	Unknown                   bool
	Err                       error
}
type Provider interface {
	Create(context.Context, Prompt, Config) ProviderResult
	Research(context.Context, ResearchInput, Config) ResearchResult
	Image(context.Context, ImagePrompt, Config) ImageResult
}
type Mock struct{}

func (*Mock) Create(_ context.Context, p Prompt, _ Config) ProviderResult {
	g := Generated{Title: "Guía de " + p.Topic + " para " + p.Audience, MetaDescription: "Conoce " + p.Topic + " con consejos prácticos para " + p.Audience + ".", Summary: "Una guía clara sobre " + p.Topic + " para " + p.Audience + ".", Content: Content{Intro: p.Topic + " puede ayudar a " + p.Audience + ". Esta guía de " + p.Business + " explica los pasos principales.", Sections: []Section{{Heading: "Por dónde empezar", Paragraphs: []string{"Define tus objetivos y considera las necesidades de tu público."}, Bullets: []string{"Identifica tus necesidades", "Compara opciones", "Mide resultados"}}}, FAQ: []FAQ{{Question: "¿Qué es " + p.Topic + "?", Answer: "Es un tema relevante para " + p.Audience + "."}}}}
	return ProviderResult{Article: g, InputTokens: 100, OutputTokens: 200}
}

type Gemini struct{ Client *http.Client }

func (g *Gemini) Create(ctx context.Context, p Prompt, c Config) ProviderResult {
	prompt := writerPrompt(p)
	schema := map[string]any{"type": "OBJECT", "properties": map[string]any{"title": map[string]any{"type": "STRING"}, "meta_description": map[string]any{"type": "STRING"}, "summary": map[string]any{"type": "STRING"}, "content": map[string]any{"type": "OBJECT", "properties": map[string]any{"intro": map[string]any{"type": "STRING"}, "sections": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "OBJECT", "properties": map[string]any{"heading": map[string]any{"type": "STRING"}, "level": map[string]any{"type": "INTEGER"}, "paragraphs": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}}, "bullets": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}}}, "required": []string{"heading", "paragraphs", "bullets"}}}, "faq": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "OBJECT", "properties": map[string]any{"question": map[string]any{"type": "STRING"}, "answer": map[string]any{"type": "STRING"}}, "required": []string{"question", "answer"}}}}, "required": []string{"intro", "sections", "faq"}}}, "required": []string{"title", "meta_description", "summary", "content"}}
	body, _ := json.Marshal(map[string]any{"contents": []any{map[string]any{"parts": []any{map[string]string{"text": prompt}}}}, "generationConfig": map[string]any{"responseMimeType": "application/json", "responseSchema": schema, "maxOutputTokens": c.MaxOutput, "temperature": 0.6, "thinkingConfig": map[string]any{"thinkingLevel": "low"}}})
	req, e := http.NewRequestWithContext(ctx, "POST", "https://generativelanguage.googleapis.com/v1beta/models/"+c.Model+":generateContent", bytes.NewReader(body))
	if e != nil {
		return ProviderResult{Err: e}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.Key)
	res, e := g.Client.Do(req)
	if e != nil {
		return ProviderResult{Unknown: true, Err: fmt.Errorf("No se pudo contactar a Gemini")}
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return ProviderResult{Err: fmt.Errorf("Gemini respondió con estado %d", res.StatusCode)}
	}
	data, e := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if e != nil {
		return ProviderResult{Err: fmt.Errorf("Respuesta Gemini inválida")}
	}
	var wire struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Usage struct {
			Prompt     int `json:"promptTokenCount"`
			Candidates int `json:"candidatesTokenCount"`
			Thoughts   int `json:"thoughtsTokenCount"`
		} `json:"usageMetadata"`
	}
	if e = json.Unmarshal(data, &wire); e != nil {
		return ProviderResult{Err: fmt.Errorf("Respuesta Gemini inválida")}
	}
	r := ProviderResult{InputTokens: wire.Usage.Prompt, OutputTokens: wire.Usage.Candidates + wire.Usage.Thoughts}
	if len(wire.Candidates) == 0 || len(wire.Candidates[0].Content.Parts) == 0 {
		r.Err = fmt.Errorf("Gemini no devolvió texto")
		return r
	}
	if e = json.Unmarshal([]byte(wire.Candidates[0].Content.Parts[0].Text), &r.Article); e != nil {
		r.Err = fmt.Errorf("Gemini devolvió JSON inválido")
		return r
	}
	return r
}
func clean(s string, max int) string {
	var b strings.Builder
	for _, r := range s {
		if r == '<' || r == '>' || r == 0 || invisible(r) {
			continue
		}
		if b.Len()+len(string(r)) > max {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}
func sanitize(c Content) (Content, error) {
	if len(c.Sections) == 0 || len(c.Sections) > 12 || len(c.FAQ) > 12 {
		return c, fmt.Errorf("Estructura inválida")
	}
	c.Intro = clean(c.Intro, 3000)
	for i := range c.Sections {
		s := &c.Sections[i]
		if len(s.Paragraphs) > 12 || len(s.Bullets) > 20 {
			return c, fmt.Errorf("Contenido extenso")
		}
		s.Heading = clean(s.Heading, 180)
		if s.Level != 3 {
			s.Level = 0
		}
		for j := range s.Paragraphs {
			s.Paragraphs[j] = clean(s.Paragraphs[j], 3000)
		}
		for j := range s.Bullets {
			s.Bullets[j] = clean(s.Bullets[j], 500)
		}
	}
	for i := range c.FAQ {
		c.FAQ[i].Question = clean(c.FAQ[i].Question, 300)
		c.FAQ[i].Answer = clean(c.FAQ[i].Answer, 1500)
	}
	return c, nil
}
func reserveAmount(c Config) int64 {
	return (20000*c.InputRate + int64(c.MaxOutput)*c.OutputRate + 999) / 1000
}
func actualCost(c Config, input, output int) int64 {
	return (int64(input)*c.InputRate + int64(output)*c.OutputRate + 999) / 1000
}
