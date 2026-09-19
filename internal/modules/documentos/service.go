package documentos

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

type Service interface {
	SubirDocumento(ctx context.Context, in SubirDocumentoInput) (*Documento, error)
	ListarMisDocumentos(ctx context.Context, idCliente uuid.UUID, idReserva *uuid.UUID) ([]Documento, error)
	ListarTodos(ctx context.Context, estado string) ([]Documento, error)
	Resolver(ctx context.Context, id, idAdmin uuid.UUID, req ResolverDocumentoRequest) (*Documento, error)

	// RequisitosReserva es el punto de integración con el módulo de reservas:
	// una reserva con servicios que exigen documento solo debe confirmarse
	// cuando PuedeConfirmarse sea true. Si idCliente no es nil, valida que la
	// reserva le pertenezca.
	RequisitosReserva(ctx context.Context, idReserva uuid.UUID, idCliente *uuid.UUID) (*RequisitosReserva, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) SubirDocumento(ctx context.Context, in SubirDocumentoInput) (*Documento, error) {
	pertenece, err := s.repo.ReservaPerteneceACliente(ctx, in.IDReserva, in.IDCliente)
	if err != nil {
		return nil, err
	}
	if !pertenece {
		return nil, utils.NotFound("reserva no encontrada")
	}

	existe, requiere, err := s.repo.ServicioRequiereDocumento(ctx, in.IDServicio)
	if err != nil {
		return nil, err
	}
	if !existe {
		return nil, utils.NotFound("servicio no encontrado")
	}
	if !requiere {
		return nil, utils.BadRequest("este servicio no requiere documento previo")
	}

	enReserva, err := s.repo.ServicioEnReserva(ctx, in.IDReserva, in.IDServicio)
	if err != nil {
		return nil, err
	}
	if !enReserva {
		return nil, utils.BadRequest("el servicio no forma parte de la reserva")
	}

	estados, err := s.repo.EstadosExistentes(ctx, in.IDReserva, in.IDServicio)
	if err != nil {
		return nil, err
	}
	for _, e := range estados {
		switch e {
		case EstadoValidado:
			return nil, utils.Conflict("el documento de este servicio ya fue validado")
		case EstadoAdjuntado:
			return nil, utils.Conflict("ya hay un documento pendiente de validación para este servicio")
		}
	}

	doc := &Documento{
		IDDocumento: uuid.New(),
		IDReserva:   in.IDReserva,
		IDServicio:  in.IDServicio,
		IDCliente:   in.IDCliente,
		RutaPDF:     in.RutaPDF,
		Estado:      EstadoAdjuntado,
		CreatedAt:   time.Now(),
	}
	if err := s.repo.Crear(ctx, doc); err != nil {
		return nil, err
	}
	return s.repo.ObtenerPorID(ctx, doc.IDDocumento)
}

func (s *service) ListarMisDocumentos(ctx context.Context, idCliente uuid.UUID, idReserva *uuid.UUID) ([]Documento, error) {
	return s.repo.ListarPorCliente(ctx, idCliente, idReserva)
}

func (s *service) ListarTodos(ctx context.Context, estado string) ([]Documento, error) {
	switch estado {
	case "", EstadoAdjuntado, EstadoValidado, EstadoRechazado:
	default:
		return nil, utils.BadRequest("estado de documento no válido")
	}
	return s.repo.ListarTodos(ctx, estado)
}

func (s *service) Resolver(ctx context.Context, id, idAdmin uuid.UUID, req ResolverDocumentoRequest) (*Documento, error) {
	respuesta := strings.TrimSpace(req.AdminRespuesta)
	switch req.Estado {
	case EstadoValidado:
	case EstadoRechazado:
		if respuesta == "" {
			return nil, utils.BadRequest("indica el motivo del rechazo")
		}
	default:
		return nil, utils.BadRequest("el estado debe ser 'validado' o 'rechazado'")
	}

	var respPtr *string
	if respuesta != "" {
		respPtr = &respuesta
	}

	ok, err := s.repo.Resolver(ctx, id, idAdmin, req.Estado, respPtr)
	if err != nil {
		return nil, err
	}
	if !ok {
		doc, err := s.repo.ObtenerPorID(ctx, id)
		if err != nil {
			return nil, err
		}
		if doc == nil {
			return nil, utils.NotFound("documento no encontrado")
		}
		return nil, utils.Conflict("el documento ya fue resuelto")
	}
	return s.repo.ObtenerPorID(ctx, id)
}

func (s *service) RequisitosReserva(ctx context.Context, idReserva uuid.UUID, idCliente *uuid.UUID) (*RequisitosReserva, error) {
	if idCliente != nil {
		pertenece, err := s.repo.ReservaPerteneceACliente(ctx, idReserva, *idCliente)
		if err != nil {
			return nil, err
		}
		if !pertenece {
			return nil, utils.NotFound("reserva no encontrada")
		}
	} else {
		existe, err := s.repo.ReservaExiste(ctx, idReserva)
		if err != nil {
			return nil, err
		}
		if !existe {
			return nil, utils.NotFound("reserva no encontrada")
		}
	}

	reqs, err := s.repo.Requisitos(ctx, idReserva)
	if err != nil {
		return nil, err
	}
	puede := true
	for _, r := range reqs {
		if !r.Cumplido {
			puede = false
			break
		}
	}
	return &RequisitosReserva{IDReserva: idReserva, Requisitos: reqs, PuedeConfirmarse: puede}, nil
}
