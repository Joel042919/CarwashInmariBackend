package pedidos

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

const (
	maxItemsPedido     = 50
	maxCantidadPorItem = 999 // detalle_pedido.cantidad es smallint
)

// El estado pagado se establece únicamente desde RF-11, dentro de la misma
// transacción que registra el pago y su comprobante interno.
type Service interface {
	CrearPedido(ctx context.Context, idCliente uuid.UUID, req CrearPedidoRequest) (*Pedido, error)
	ListarMisPedidos(ctx context.Context, idCliente uuid.UUID) ([]Pedido, error)
	ListarTodos(ctx context.Context, estado string) ([]Pedido, error)
	CancelarMiPedido(ctx context.Context, idCliente, id uuid.UUID) (*Pedido, error)
	CambiarEstado(ctx context.Context, id uuid.UUID, nuevo string) (*Pedido, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func estadoValido(e string) bool {
	switch e {
	case EstadoRegistrado, EstadoPagado, EstadoPreparando, EstadoEntregado, EstadoCancelado:
		return true
	}
	return false
}

func (s *service) CrearPedido(ctx context.Context, idCliente uuid.UUID, req CrearPedidoRequest) (*Pedido, error) {
	if len(req.Items) == 0 {
		return nil, utils.BadRequest("el pedido debe incluir al menos un producto")
	}
	if len(req.Items) > maxItemsPedido {
		return nil, utils.BadRequest("el pedido supera el máximo de productos permitidos")
	}

	// Consolida líneas repetidas del mismo producto.
	acumulado := map[uuid.UUID]int{}
	orden := []uuid.UUID{}
	for _, it := range req.Items {
		if it.IDProducto == uuid.Nil {
			return nil, utils.BadRequest("producto inválido")
		}
		if it.Cantidad <= 0 {
			return nil, utils.BadRequest("la cantidad de cada producto debe ser mayor a 0")
		}
		if _, ya := acumulado[it.IDProducto]; !ya {
			orden = append(orden, it.IDProducto)
		}
		acumulado[it.IDProducto] += it.Cantidad
	}
	items := make([]ItemPedido, 0, len(orden))
	for _, id := range orden {
		if acumulado[id] > maxCantidadPorItem {
			return nil, utils.BadRequest("la cantidad máxima por producto es 999")
		}
		items = append(items, ItemPedido{IDProducto: id, Cantidad: acumulado[id]})
	}

	var obs *string
	if t := strings.TrimSpace(req.Observaciones); t != "" {
		obs = &t
	}
	return s.repo.CrearPedido(ctx, idCliente, items, obs)
}

func (s *service) ListarMisPedidos(ctx context.Context, idCliente uuid.UUID) ([]Pedido, error) {
	return s.repo.ListarPorCliente(ctx, idCliente)
}

func (s *service) ListarTodos(ctx context.Context, estado string) ([]Pedido, error) {
	if estado != "" && !estadoValido(estado) {
		return nil, utils.BadRequest("estado de pedido no válido")
	}
	return s.repo.ListarTodos(ctx, estado)
}

func (s *service) CancelarMiPedido(ctx context.Context, idCliente, id uuid.UUID) (*Pedido, error) {
	if err := s.repo.CambiarEstado(ctx, id, EstadoCancelado, &idCliente); err != nil {
		return nil, err
	}
	return s.repo.ObtenerPorID(ctx, id)
}

func (s *service) CambiarEstado(ctx context.Context, id uuid.UUID, nuevo string) (*Pedido, error) {
	if strings.TrimSpace(nuevo) == EstadoPagado {
		return nil, utils.Conflict("registra el cobro desde el módulo de pagos; el pedido se marcará como pagado automáticamente")
	}
	if !estadoValido(nuevo) || nuevo == EstadoRegistrado {
		return nil, utils.BadRequest("estado de pedido no válido")
	}
	if err := s.repo.CambiarEstado(ctx, id, nuevo, nil); err != nil {
		return nil, err
	}
	return s.repo.ObtenerPorID(ctx, id)
}
