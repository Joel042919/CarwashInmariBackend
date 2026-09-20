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
	Activo            bool      `json:"activo"`
	FechaCese         *string   `json:"fecha_cese,omitempty"`
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
	Disponible *bool `json:"disponible"`
}

// Actualizar no permite cambiar el rol ni la contraseña desde la ficha laboral.
type ActualizarTrabajadorInput struct {
	Nombre            string  `json:"nombre"`
	Apellido          string  `json:"apellido"`
	Correo            string  `json:"correo"`
	Telefono          *string `json:"telefono"`
	DNI               string  `json:"dni"`
	FechaContratacion string  `json:"fecha_contratacion"`
}
