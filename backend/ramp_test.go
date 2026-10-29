package main

import (
	"testing"
	"time"
)

// El ritmo elegido aplica hasta que termina el arranque de 2 semanas; antes es 1 por semana.
// Un blog con historia (5+ publicados) se salta el arranque. Nunca más de 5 por semana.
func TestPerWeekAt(t *testing.T) {
	created := time.Now()
	if n := perWeekAt(created, 3, false, created.Add(24*time.Hour)); n != 1 {
		t.Fatalf("en el arranque: %d", n)
	}
	if n := perWeekAt(created, 5, true, created.Add(24*time.Hour)); n != 5 {
		t.Fatalf("blog con historia, sin arranque: %d", n)
	}
	if n := perWeekAt(created, 3, false, created.Add(15*24*time.Hour)); n != 3 {
		t.Fatalf("pasado el arranque: %d", n)
	}
	if n := perWeekAt(created, 9, false, created.Add(30*24*time.Hour)); n != 5 {
		t.Fatalf("tope de 5: %d", n)
	}
}
