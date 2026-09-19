package vehiculos

import (
	"time"

	"github.com/google/uuid"
)

type Vehiculo struct {
	IDVehiculo   uuid.UUID `json:"id_vehiculo"`
	IDCliente    uuid.UUID `json:"id_cliente"`
	Placa        string    `json:"placa"`
	Marca        string    `json:"marca"`
	Modelo       string    `json:"modelo"`
	Color        *string   `json:"color,omitempty"`
	Anio         *int      `json:"anio,omitempty"`
	TipoVehiculo *string   `json:"tipo_vehiculo,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type VehiculoInput struct {
	Placa        string  `json:"placa"`
	Marca        string  `json:"marca"`
	Modelo       string  `json:"modelo"`
	Color        *string `json:"color"`
	Anio         *int    `json:"anio"`
	TipoVehiculo *string `json:"tipo_vehiculo"`
}
