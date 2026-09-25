package reservas

import "sort"

// Rango es un intervalo en minutos desde las 00:00 (inicio incluido, fin excluido).
type Rango struct {
	Inicio int
	Fin    int
}

func (r Rango) seCruza(otro Rango) bool {
	return r.Inicio < otro.Fin && r.Fin > otro.Inicio
}

// pasoSlotMin es la separación entre horas de inicio ofrecidas al cliente.
const pasoSlotMin = 5

// GenerarSlots devuelve los rangos de `duracion` minutos que caben dentro de los
// tramos de atención sin cruzarse con ningún rango ocupado y que empiezan a partir
// de `minInicio`. Las horas de inicio salen cada 5 minutos desde el comienzo de cada
// tramo y, además, justo al terminar cada reserva ocupada, para no perder huecos.
func GenerarSlots(tramos, ocupados []Rango, duracion, minInicio int) []Rango {
	if duracion <= 0 {
		return nil
	}
	slots := []Rango{}
	for _, t := range tramos {
		candidatos := map[int]bool{}
		for s := t.Inicio; s+duracion <= t.Fin; s += pasoSlotMin {
			candidatos[s] = true
		}
		for _, o := range ocupados {
			if o.Fin >= t.Inicio && o.Fin+duracion <= t.Fin {
				candidatos[o.Fin] = true
			}
		}

		inicios := make([]int, 0, len(candidatos))
		for s := range candidatos {
			inicios = append(inicios, s)
		}
		sort.Ints(inicios)

		for _, s := range inicios {
			if s < minInicio {
				continue
			}
			cand := Rango{Inicio: s, Fin: s + duracion}
			libre := true
			for _, o := range ocupados {
				if cand.seCruza(o) {
					libre = false
					break
				}
			}
			if libre {
				slots = append(slots, cand)
			}
		}
	}
	return slots
}

// dentroDeTramos indica si el rango cabe completo en alguno de los tramos de atención.
func dentroDeTramos(r Rango, tramos []Rango) bool {
	for _, t := range tramos {
		if r.Inicio >= t.Inicio && r.Fin <= t.Fin {
			return true
		}
	}
	return false
}
