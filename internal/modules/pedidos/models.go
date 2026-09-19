package pedidos

import (
	"time"

	"github.com/google/uuid"
)

const (
	EstadoRegistrado = "registrado"
	EstadoPagado     = "pagado"
	EstadoPreparando = "preparando"
	EstadoEntregado  = "entregado"
	EstadoCancelado  = "cancelado"
)

type DetallePedido struct {
	IDDetalle      uuid.UUID `json:"id_detalle"`
	IDProducto     uuid.UUID `json:"id_producto"`
	NombreProducto string    `json:"nombre_producto"`
	Cantidad       int       `json:"cantidad"`
	PrecioUnitario float64   `json:"precio_unitario"`
	Subtotal       float64   `json:"subtotal"`
}

type Pedido struct {
	IDPedido      uuid.UUID       `json:"id_pedido"`
	IDCliente     uuid.UUID       `json:"id_cliente"`
	NombreCliente string          `json:"nombre_cliente,omitempty"`
	CorreoCliente string          `json:"correo_cliente,omitempty"`
	Estado        string          `json:"estado"`
	Total         float64         `json:"total"`
	Observaciones *string         `json:"observaciones,omitempty"`
	FechaRegistro time.Time       `json:"fecha_registro"`
	FechaEntrega  *time.Time      `json:"fecha_entrega,omitempty"`
	Detalle       []DetallePedido `json:"detalle"`
}

type ItemPedido struct {
	IDProducto uuid.UUID `json:"id_producto"`
	Cantidad   int       `json:"cantidad"`
}

type CrearPedidoRequest struct {
	Items         []ItemPedido `json:"items"`
	Observaciones string       `json:"observaciones"`
}

type CambiarEstadoRequest struct {
	Estado string `json:"estado"`
}
