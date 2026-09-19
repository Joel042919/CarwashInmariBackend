package trabajadores

import "github.com/google/uuid"

type Trabajador struct {
	IDUsuario         uuid.UUID `json:"id_usuario"`
	Nombre            string    `json:"nombre"`
	Apellido          string    `json:"apellido"`
	Correo            string    `json:"correo"`
	Telefono          *string   `json:"telefono,omitempty"`
	DNI               string    `json:"dni"`
	FechaContratacion string    `json:"fecha_contratacion"`
	Disponible        bool      `json:"disponible"`
}

type CrearTrabajadorInput struct {
	Nombre            string  `json:"nombre"`
	Apellido          string  `json:"apellido"`
	Correo            string  `json:"correo"`
	Telefono          *string `json:"telefono"`
	Contrasena        string  `json:"contrasena"`
	DNI               string  `json:"dni"`
	FechaContratacion string  `json:"fecha_contratacion"` // AAAA-MM-DD; si va vacía se usa hoy
}

type DisponibilidadInput struct {
	Disponible bool `json:"disponible"`
}
