package reclamos

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	CrearReclamo(ctx context.Context, input CrearReclamoInput) (*Reclamo, error)
	ListarMisReclamos(ctx context.Context, idCliente uuid.UUID) ([]Reclamo, error)
	ListarTodos(ctx context.Context) ([]Reclamo, error)
	ResponderReclamo(ctx context.Context, idReclamo, idAdmin uuid.UUID, req ResponderReclamoRequest) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CrearReclamo(ctx context.Context, input CrearReclamoInput) (*Reclamo, error) {
	input.Asunto = strings.TrimSpace(input.Asunto)
	input.Descripcion = strings.TrimSpace(input.Descripcion)

	if input.Asunto == "" || input.Descripcion == "" {
		return nil, errors.New("el asunto y la descripción son obligatorios")
	}

	idReclamo := uuid.New()
	now := time.Now()

	var evidencias []EvidenciaReclamo
	for _, ev := range input.Evidencias {
		evidencias = append(evidencias, EvidenciaReclamo{
			IDEvidencia: uuid.New(),
			IDReclamo:   idReclamo,
			RutaArchivo: ev.RutaArchivo,
			Descripcion: ev.Descripcion,
			CreatedAt:   now,
		})
	}

	nuevoReclamo := &Reclamo{
		IDReclamo:     idReclamo,
		IDCliente:     input.IDCliente,
		IDAtencion:    input.IDAtencion,
		IDPago:        input.IDPago,
		IDPedido:      input.IDPedido,
		Asunto:        input.Asunto,
		Descripcion:   input.Descripcion,
		Estado:        EstadoRegistrado,
		FechaRegistro: now,
		Evidencias:    evidencias,
	}

	if err := s.repo.CrearReclamoConEvidenciasTx(ctx, nuevoReclamo); err != nil {
		return nil, errors.New("error al guardar el reclamo: " + err.Error())
	}

	return nuevoReclamo, nil
}

func (s *service) ListarMisReclamos(ctx context.Context, idCliente uuid.UUID) ([]Reclamo, error) {
	return s.repo.ListarPorCliente(ctx, idCliente)
}

func (s *service) ListarTodos(ctx context.Context) ([]Reclamo, error) {
	return s.repo.ListarTodos(ctx)
}

func (s *service) ResponderReclamo(ctx context.Context, idReclamo, idAdmin uuid.UUID, req ResponderReclamoRequest) error {
	req.RespuestaAdmin = strings.TrimSpace(req.RespuestaAdmin)
	if req.RespuestaAdmin == "" {
		return errors.New("la respuesta del administrador no puede estar vacía")
	}

	switch req.NuevoEstado {
	case EstadoEnRevision, EstadoRespondido, EstadoCerrado:
	default:
		return errors.New("estado de respuesta no válido (use 'en_revision', 'respondido' o 'cerrado')")
	}

	return s.repo.ResponderReclamo(ctx, idReclamo, idAdmin, req.RespuestaAdmin, req.NuevoEstado)
}
