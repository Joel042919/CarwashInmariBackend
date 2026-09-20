package atenciones

import (
	"carwashinmaribackend/internal/utils"
	"context"
	"github.com/google/uuid"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func validarTransicion(actual, nuevo, reserva string) error {
	if reserva != "confirmada" {
		return utils.Conflict("la reserva debe estar confirmada antes de operar la atención")
	}
	if (actual == Programada && nuevo == EnProceso) || (actual == EnProceso && nuevo == Finalizada) {
		return nil
	}
	return utils.Conflict("transición de atención no permitida")
}

func (s *Service) Listar(ctx context.Context, actor utils.CustomClaims, f Filtro) ([]Atencion, error) {
	switch f.Estado {
	case "", Programada, EnProceso, Finalizada, "en_pausa", "entregada":
	default:
		return nil, utils.BadRequest("estado de atención inválido")
	}
	return s.repo.Listar(ctx, actor, f)
}

func (s *Service) CambiarEstado(ctx context.Context, actor utils.CustomClaims, id uuid.UUID, nuevo string) error {
	if actor.Rol != "administrador" && actor.Rol != "trabajador" {
		return &utils.AppError{Status: 403, Msg: "No puedes modificar atenciones"}
	}
	if nuevo != EnProceso && nuevo != Finalizada {
		return utils.BadRequest("estado de destino inválido")
	}
	return s.repo.CambiarEstado(ctx, actor, id, nuevo)
}

func (s *Service) Historial(ctx context.Context, actor utils.CustomClaims, id uuid.UUID) ([]Evento, error) {
	return s.repo.Historial(ctx, actor, id)
}
func (s *Service) Rendimiento(ctx context.Context, actor utils.CustomClaims, id uuid.UUID, rango utils.RangoFechas) (*Rendimiento, error) {
	if actor.Rol != "administrador" && !(actor.Rol == "trabajador" && actor.UserID == id) {
		return nil, &utils.AppError{Status: 403, Msg: "No puedes consultar ese rendimiento"}
	}
	return s.repo.Rendimiento(ctx, actor.SedeID, id, rango)
}
