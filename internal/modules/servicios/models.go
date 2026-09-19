package servicios

import (
	"time"

	"github.com/google/uuid"
)

type Categoria struct {
	IDCategoria uuid.UUID `json:"id_categoria_servicio"`
	Nombre      string    `json:"categoria_servicio"`
}

type CategoriaInput struct {
	Nombre string `json:"categoria_servicio"`
}

type Servicio struct {
	IDServicio          uuid.UUID `json:"id_servicio"`
	Nombre              string    `json:"nombre"`
	Descripcion         *string   `json:"descripcion,omitempty"`
	Precio              float64   `json:"precio"`
	DuracionEstimadaMin int       `json:"duracion_estimada_min"`
	RequiereDocumento   bool      `json:"requiere_documento"`
	Disponible          bool      `json:"disponible"`
	IDCategoria         uuid.UUID `json:"id_categoria"`
	Categoria           string    `json:"categoria,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

type ServicioInput struct {
	Nombre              string    `json:"nombre"`
	Descripcion         *string   `json:"descripcion"`
	Precio              float64   `json:"precio"`
	DuracionEstimadaMin int       `json:"duracion_estimada_min"`
	RequiereDocumento   bool      `json:"requiere_documento"`
	Disponible          *bool     `json:"disponible"`
	IDCategoria         uuid.UUID `json:"id_categoria"`
}

type DisponibilidadInput struct {
	Disponible bool `json:"disponible"`
}
