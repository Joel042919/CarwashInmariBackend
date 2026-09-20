package atenciones

import (
	"carwashinmaribackend/internal/utils"
	"github.com/google/uuid"
	"time"
)

const (
	Programada = "programada"
	EnProceso  = "en_proceso"
	Finalizada = "finalizada"
)

type Filtro struct {
	Trabajador *uuid.UUID
	Estado     string
	Rango      utils.RangoFechas
}

type Servicio struct {
	Nombre   string `json:"nombre"`
	Cantidad int    `json:"cantidad"`
}

type Atencion struct {
	ID            uuid.UUID  `json:"id_atencion"`
	IDReserva     uuid.UUID  `json:"id_reserva"`
	Estado        string     `json:"estado"`
	EstadoReserva string     `json:"estado_reserva"`
	Fecha         string     `json:"fecha"`
	HoraInicio    string     `json:"hora_inicio"`
	HoraFin       string     `json:"hora_fin"`
	Placa         string     `json:"placa"`
	Cliente       string     `json:"cliente"`
	InicioReal    *time.Time `json:"fecha_inicio_real"`
	FinReal       *time.Time `json:"fecha_fin_real"`
	Servicios     []Servicio `json:"servicios"`
}

type Evento struct {
	ID          uuid.UUID `json:"id_historial"`
	Estado      string    `json:"estado"`
	Fecha       time.Time `json:"fecha_cambio"`
	Responsable string    `json:"responsable"`
	Comentario  *string   `json:"comentario"`
}

type Rendimiento struct {
	Atenciones       int      `json:"atenciones_finalizadas"`
	Servicios        int      `json:"servicios_en_equipo"`
	DuracionPromedio *float64 `json:"duracion_promedio_min"`
}

type RendimientoEquipo struct {
	IDTrabajador     uuid.UUID `json:"id_trabajador"`
	Trabajador       string    `json:"trabajador"`
	Atenciones       int       `json:"atenciones_finalizadas"`
	Servicios        int       `json:"servicios_en_equipo"`
	DuracionPromedio *float64  `json:"duracion_promedio_min"`
}
