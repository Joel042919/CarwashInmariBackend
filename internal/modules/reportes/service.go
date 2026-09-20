package reportes

import (
	"context"
	"time"

	"carwashinmaribackend/internal/utils"
	"github.com/google/uuid"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func normalizarRango(r utils.RangoFechas, ahora time.Time) (utils.RangoFechas, error) {
	local := ahora.In(utils.ZonaNegocio)
	hoy := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, utils.ZonaNegocio)
	if r.Desde == nil && r.Hasta == nil {
		desde := hoy.AddDate(0, 0, -29)
		hasta := hoy.AddDate(0, 0, 1)
		r.Desde = &desde
		r.Hasta = &hasta
	}
	if r.Desde == nil {
		desde := r.Hasta.AddDate(0, 0, -30)
		r.Desde = &desde
	}
	if r.Hasta == nil {
		hasta := hoy.AddDate(0, 0, 1)
		r.Hasta = &hasta
	}
	if !r.Desde.Before(*r.Hasta) {
		return r, utils.BadRequest("el periodo debe contener al menos un día")
	}
	if r.Hasta.Sub(*r.Desde) > 366*24*time.Hour {
		return r, utils.BadRequest("el periodo máximo de consulta es 366 días")
	}
	return r, nil
}

func (s *Service) Consultar(ctx context.Context, sede uuid.UUID, rango utils.RangoFechas) (*Dashboard, error) {
	rango, err := normalizarRango(rango, time.Now())
	if err != nil {
		return nil, err
	}
	d, err := s.repo.Consultar(ctx, sede, rango)
	if err != nil {
		return nil, err
	}
	d.Periodo = Periodo{Desde: rango.Desde.In(utils.ZonaNegocio).Format("2006-01-02"), Hasta: rango.Hasta.AddDate(0, 0, -1).In(utils.ZonaNegocio).Format("2006-01-02")}
	return d, nil
}
