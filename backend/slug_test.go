package main

import "testing"

func TestSlugSinAcentos(t *testing.T) {
	cases := map[string]string{
		"Cómo digitalizar el expediente clínico en México":  "como-digitalizar-el-expediente-clinico-en-mexico",
		"¿Qué es un sistema a la medida y cuándo conviene?": "que-es-un-sistema-a-la-medida-y-cuando-conviene",
		"Diseño para niños y pingüinos":                     "diseno-para-ninos-y-pinguinos",
		"???":                                               "articulo",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, quería %q", in, got, want)
		}
	}
}
