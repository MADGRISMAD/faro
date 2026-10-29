package main

import (
	"strings"
	"testing"
)

func TestExtractPaletteTakesBrandColors(t *testing.T) {
	html := `<html><head><meta name="theme-color" content="#c2185b"><style>:root{--accent:#c2185b;--accent-d:#9d1049;--ink:#1a1a1a;--bg:#fcfafb}
	.btn{background:#c2185b;color:#fff}.x{border:1px solid #eadde3;color:#666}.tag{background:rgb(32, 120, 200)}.hover{background:#c41a5d}</style></head></html>`
	p := extractPalette(html, []string{`.card{color:#0f8a6a}`})
	if len(p) == 0 || p[0] != "#c2185b" {
		t.Fatalf("el color de marca va primero: %v", p)
	}
	all := strings.Join(p, " ")
	for _, gris := range []string{"#ffffff", "#1a1a1a", "#fcfafb", "#666666", "#eadde3"} {
		if strings.Contains(all, gris) {
			t.Errorf("%s es blanco, negro o gris: %v", gris, p)
		}
	}
	if strings.Contains(all, "#c41a5d") {
		t.Errorf("un tono casi igual al principal cuenta como el mismo: %v", p)
	}
	for _, want := range []string{"#2078c8", "#0f8a6a"} {
		if !strings.Contains(all, want) {
			t.Errorf("faltó %s: %v", want, p)
		}
	}
	if p := extractPalette(`<style>body{color:#000;background:#fff}</style>`, nil); len(p) != 0 {
		t.Errorf("sin colores de marca: %v", p)
	}
}

func TestImagePromptUsesPaletteAndMemphis(t *testing.T) {
	p := imageText(ImagePrompt{Title: "X", Business: "Y", Audience: "Z", Palette: []string{"#c2185b", "#2078c8"}})
	if !strings.Contains(p, "Memphis corporativo") || !strings.Contains(p, "#c2185b, #2078c8") || !strings.Contains(p, "Ningún otro color") {
		t.Errorf("prompt: %s", p)
	}
	if strings.Contains(imageText(ImagePrompt{Title: "X", Style: "photo"}), "Ningún otro color") {
		t.Error("sin paleta guardada no se impone una lista de colores")
	}
}
