package espacios

import "testing"

func TestValidarHorariosOK(t *testing.T) {
	in := []Horario{
		{DiaSemana: 1, HoraInicio: "08:00", HoraFin: "12:00"},
		{DiaSemana: 1, HoraInicio: "14:00", HoraFin: "18:00"},
		{DiaSemana: 2, HoraInicio: "9:30", HoraFin: "13:00"},
	}
	out, err := ValidarHorarios(in)
	if err != nil {
		t.Fatal(err)
	}
	if out[2].HoraInicio != "09:30" {
		t.Fatalf("esperado 09:30, got %s", out[2].HoraInicio)
	}
}

func TestValidarHorariosSolape(t *testing.T) {
	_, err := ValidarHorarios([]Horario{
		{DiaSemana: 1, HoraInicio: "08:00", HoraFin: "12:00"},
		{DiaSemana: 1, HoraInicio: "11:00", HoraFin: "14:00"},
	})
	if err == nil {
		t.Fatal("debería rechazar tramos solapados")
	}
}

func TestValidarHorariosFinAntesDeInicio(t *testing.T) {
	_, err := ValidarHorarios([]Horario{
		{DiaSemana: 3, HoraInicio: "18:00", HoraFin: "08:00"},
	})
	if err == nil {
		t.Fatal("debería rechazar fin <= inicio")
	}
}
