package espacios

import "github.com/google/uuid"

// Horario es un tramo de atención de un espacio en un día de la semana (0 = domingo ... 6 = sábado).
type Horario struct {
	DiaSemana  int    `json:"dia_semana"`
	HoraInicio string `json:"hora_inicio"` // HH:MM
	HoraFin    string `json:"hora_fin"`    // HH:MM
}

type Espacio struct {
	IDEspacio uuid.UUID `json:"id_espacio"`
	Codigo    string    `json:"codigo"`
	Activo    bool      `json:"activo"`
	Horarios  []Horario `json:"horarios"`
}

type EspacioInput struct {
	Codigo   string    `json:"codigo"`
	Activo   *bool     `json:"activo"`
	Horarios []Horario `json:"horarios"` // opcional al crear: define el horario semanal de una vez
}

type HorariosInput struct {
	Horarios []Horario `json:"horarios"`
}
