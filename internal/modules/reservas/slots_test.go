package reservas

import "testing"

func TestGenerarSlotsBasico(t *testing.T) {
	tramos := []Rango{{Inicio: 8 * 60, Fin: 12 * 60}} // 08:00-12:00
	ocupados := []Rango{}
	got := GenerarSlots(tramos, ocupados, 60, 0)
	if len(got) != 37 || got[0] != (Rango{Inicio: 480, Fin: 540}) || got[1] != (Rango{Inicio: 485, Fin: 545}) || got[len(got)-1] != (Rango{Inicio: 660, Fin: 720}) {
		t.Fatalf("cuadrícula de 5 minutos inesperada: %v", got)
	}
}

func TestGenerarSlotsRespetaOcupacion(t *testing.T) {
	tramos := []Rango{{Inicio: 8 * 60, Fin: 11 * 60}}
	ocupados := []Rango{{Inicio: 9 * 60, Fin: 10 * 60}} // 09:00-10:00
	got := GenerarSlots(tramos, ocupados, 60, 0)

	for _, s := range got {
		if s.seCruza(ocupados[0]) {
			t.Fatalf("slot %v se cruza con ocupación", s)
		}
	}
	// Debe existir el hueco justo al terminar la ocupación (10:00-11:00).
	found := false
	for _, s := range got {
		if s.Inicio == 10*60 && s.Fin == 11*60 {
			found = true
		}
	}
	if !found {
		t.Fatalf("falta el slot 10:00-11:00, got=%v", got)
	}
}

func TestGenerarSlotsMinInicio(t *testing.T) {
	tramos := []Rango{{Inicio: 8 * 60, Fin: 10 * 60}}
	got := GenerarSlots(tramos, nil, 60, 9*60)
	for _, s := range got {
		if s.Inicio < 9*60 {
			t.Fatalf("slot %v empieza antes del mínimo", s)
		}
	}
}

func TestDentroDeTramos(t *testing.T) {
	tramos := []Rango{{Inicio: 8 * 60, Fin: 12 * 60}, {Inicio: 14 * 60, Fin: 18 * 60}}
	if !dentroDeTramos(Rango{Inicio: 9 * 60, Fin: 10 * 60}, tramos) {
		t.Fatal("debería caber en el tramo de la mañana")
	}
	if dentroDeTramos(Rango{Inicio: 11 * 60, Fin: 13 * 60}, tramos) {
		t.Fatal("no debería cruzar el hueco del mediodía")
	}
}
