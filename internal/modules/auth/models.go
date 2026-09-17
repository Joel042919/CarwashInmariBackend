package auth

import (
	"time"

	"github.com/google/uuid"
)

type Usuario struct {
	IDUsuario      uuid.UUID `json:"id_usuario"`
	IDSede         uuid.UUID `json:"id_sede"`
	Nombre         string    `json:"nombre"`
	Apellido       string    `json:"apellido"`
	Correo         string    `json:"correo"`
	Telefono       *string   `json:"telefono,omitempty"`
	ContrasenaHash string    `json:"-"` // Nunca exponer en JSON
	IDRol          uuid.UUID `json:"id_rol"`
	RolNombre      string    `json:"rol,omitempty"`
	Activo         bool      `json:"activo"`
	FechaRegistro  time.Time `json:"fecha_registro"`
}
type LoginRequest struct {
	Correo     string `json:"correo"`
	Contrasena string `json:"contrasena"`
}

type RegisterRequest struct {
	IDSede     uuid.UUID `json:"id_sede"`
	Nombre     string    `json:"nombre"`
	Apellido   string    `json:"apellido"`
	Correo     string    `json:"correo"`
	Telefono   *string   `json:"telefono,omitempty"`
	Contrasena string    `json:"contrasena"`
	DNI        *string   `json:"dni,omitempty"`
}

type AuthResponse struct {
	Token   string  `json:"token"`
	Usuario Usuario `json:"usuario"`
}
