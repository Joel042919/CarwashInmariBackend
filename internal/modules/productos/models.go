package productos

import "github.com/google/uuid"

type Categoria struct {
	IDCategoria uuid.UUID `json:"id_categoria"`
	Nombre      string    `json:"nombre"`
	Descripcion *string   `json:"descripcion,omitempty"`
	Activo      bool      `json:"activo"`
}

type CategoriaInput struct {
	Nombre      string  `json:"nombre"`
	Descripcion *string `json:"descripcion"`
	Activo      *bool   `json:"activo"`
}

type Producto struct {
	IDProducto  uuid.UUID `json:"id_producto"`
	IDCategoria uuid.UUID `json:"id_categoria"`
	Categoria   string    `json:"categoria,omitempty"`
	Codigo      string    `json:"codigo"`
	Nombre      string    `json:"nombre"`
	Descripcion *string   `json:"descripcion,omitempty"`
	PrecioVenta float64   `json:"precio_venta"`
	Stock       int       `json:"stock"`
	StockMinimo int       `json:"stock_minimo"`
	Activo      bool      `json:"activo"`
	StockBajo   bool      `json:"stock_bajo"`
}

type ProductoInput struct {
	IDCategoria uuid.UUID `json:"id_categoria"`
	Codigo      string    `json:"codigo"`
	Nombre      string    `json:"nombre"`
	Descripcion *string   `json:"descripcion"`
	PrecioVenta float64   `json:"precio_venta"`
	Stock       int       `json:"stock"`
	StockMinimo *int      `json:"stock_minimo"`
	Activo      *bool     `json:"activo"`
}

// AjusteStockInput suma (o resta si es negativo) unidades al stock actual.
type AjusteStockInput struct {
	Cantidad int `json:"cantidad"`
}

type FiltroProductos struct {
	SoloActivos bool
	IDCategoria *uuid.UUID
	Busqueda    string
	SoloBajos   bool
}
