package clientes

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type Service interface {
	ObtenerMiPerfil(ctx context.Context, idUsuario uuid.UUID) (*ClientePerfil, error)
	ActualizarMiPerfil(ctx context.Context, idUsuario uuid.UUID, req UpdateClienteRequest) error
	ObtenerHistorialUnificado(ctx context.Context, idUsuario uuid.UUID) (*HistorialClienteUnificado, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) ObtenerMiPerfil(ctx context.Context, idUsuario uuid.UUID) (*ClientePerfil, error) {
	perfil, err := s.repo.ObtenerPerfil(ctx, idUsuario)
	if err != nil {
		return nil, err
	}
	if perfil == nil {
		return nil, errors.New("perfil de cliente no encontrado")
	}
	return perfil, nil
}

func (s *service) ActualizarMiPerfil(ctx context.Context, idUsuario uuid.UUID, req UpdateClienteRequest) error {
	req.Nombre = strings.TrimSpace(req.Nombre)
	req.Apellido = strings.TrimSpace(req.Apellido)

	if req.Nombre == "" || req.Apellido == "" {
		return errors.New("el nombre y el apellido no pueden estar vacíos")
	}

	return s.repo.ActualizarPerfil(ctx, idUsuario, req)
}

func (s *service) ObtenerHistorialUnificado(ctx context.Context, idUsuario uuid.UUID) (*HistorialClienteUnificado, error) {
	reservas, err := s.repo.ListarReservasPorCliente(ctx, idUsuario)
	if err != nil {
		return nil, errors.New("error al obtener reservas: " + err.Error())
	}

	atenciones, err := s.repo.ListarAtencionesPorCliente(ctx, idUsuario)
	if err != nil {
		return nil, errors.New("error al obtener atenciones: " + err.Error())
	}

	pagos, err := s.repo.ListarPagosPorCliente(ctx, idUsuario)
	if err != nil {
		return nil, errors.New("error al obtener pagos: " + err.Error())
	}

	pedidos, err := s.repo.ListarPedidosPorCliente(ctx, idUsuario)
	if err != nil {
		return nil, errors.New("error al obtener compras/pedidos: " + err.Error())
	}

	// Inicializar slices vacíos en caso de no haber registros (para evitar arrays "null" en JSON)
	if reservas == nil {
		reservas = []ReservaHistorial{}
	}
	if atenciones == nil {
		atenciones = []AtencionHistorial{}
	}
	if pagos == nil {
		pagos = []PagoHistorial{}
	}
	if pedidos == nil {
		pedidos = []PedidoHistorial{}
	}

	return &HistorialClienteUnificado{
		Reservas:   reservas,
		Atenciones: atenciones,
		Pagos:      pagos,
		Pedidos:    pedidos,
	}, nil
}
