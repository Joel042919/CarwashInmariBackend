package evidencias

import (
	"github.com/google/uuid"
	"time"
)

type Evidencia struct {
	IDEvidencia   uuid.UUID `json:"id_evidencia"`
	IDAtencion    uuid.UUID `json:"id_atencion"`
	IDVehiculo    uuid.UUID `json:"id_vehiculo"`
	RegistradoPor uuid.UUID `json:"registrado_por"`
	Tipo          string    `json:"tipo"`
	Descripcion   *string   `json:"descripcion,omitempty"`
	RutaFoto      string    `json:"ruta_foto"`
	CreatedAt     time.Time `json:"created_at"`
}
