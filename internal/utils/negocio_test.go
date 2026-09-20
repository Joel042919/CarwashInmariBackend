package utils

import "testing"

func TestRangoFechasLima(t *testing.T) {
	r, err := ParseRangoFechas("2026-09-19", "2026-09-19")
	if err != nil {
		t.Fatal(err)
	}
	if r.Desde.UTC().Format("2006-01-02T15:04:05Z") != "2026-09-19T05:00:00Z" || r.Hasta.Sub(*r.Desde).Hours() != 24 {
		t.Fatal("límites de día incorrectos")
	}
	for _, par := range [][2]string{{"2026-02-30", ""}, {"2026-09-20", "2026-09-19"}, {"ayer", "hoy"}} {
		if _, err := ParseRangoFechas(par[0], par[1]); err == nil {
			t.Fatalf("rango inválido aceptado: %v", par)
		}
	}
}
