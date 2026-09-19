package espacios

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

const maxTramosPorDia = 6

type Service interface {
	Listar(ctx context.Context, soloActivos bool) ([]Espacio, error)
	Crear(ctx context.Context, in EspacioInput) (*Espacio, error)
	Actualizar(ctx context.Context, id uuid.UUID, in EspacioInput) (*Espacio, error)
	ReemplazarHorarios(ctx context.Context, id uuid.UUID, in HorariosInput) (*Espacio, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Listar(ctx context.Context, soloActivos bool) ([]Espacio, error) {
	return s.repo.Listar(ctx, soloActivos)
}

func validarCodigo(in *EspacioInput) error {
	in.Codigo = strings.ToUpper(strings.TrimSpace(in.Codigo))
	if in.Codigo == "" || len(in.Codigo) > 20 {
		return utils.BadRequest("el código del espacio es obligatorio (máximo 20 caracteres)")
	}
	return nil
}

func (s *service) Crear(ctx context.Context, in EspacioInput) (*Espacio, error) {
	if err := validarCodigo(&in); err != nil {
		return nil, err
	}
	activo := true
	if in.Activo != nil {
		activo = *in.Activo
	}
	e := &Espacio{IDEspacio: uuid.New(), Codigo: in.Codigo, Activo: activo, Horarios: []Horario{}}
	if err := s.repo.Crear(ctx, e); err != nil {
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe un espacio con ese código")
		}
		return nil, err
	}
	return e, nil
}

func (s *service) Actualizar(ctx context.Context, id uuid.UUID, in EspacioInput) (*Espacio, error) {
	if err := validarCodigo(&in); err != nil {
		return nil, err
	}
	actual, err := s.repo.Obtener(ctx, id)
	if err != nil {
		return nil, err
	}
	if actual == nil {
		return nil, utils.NotFound("espacio no encontrado")
	}
	activo := actual.Activo
	if in.Activo != nil {
		activo = *in.Activo
	}
	if _, err := s.repo.Actualizar(ctx, &Espacio{IDEspacio: id, Codigo: in.Codigo, Activo: activo}); err != nil {
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe un espacio con ese código")
		}
		return nil, err
	}
	return s.repo.Obtener(ctx, id)
}

// ValidarHorarios comprueba días, formato, que fin > inicio y que no haya tramos
// solapados dentro del mismo día. Normaliza las horas a "HH:MM".
func ValidarHorarios(horarios []Horario) ([]Horario, error) {
	porDia := map[int][][2]int{}
	normalizados := make([]Horario, 0, len(horarios))

	for _, h := range horarios {
		if h.DiaSemana < 0 || h.DiaSemana > 6 {
			return nil, utils.BadRequest("el día de la semana debe estar entre 0 (domingo) y 6 (sábado)")
		}
		ini, err := utils.ParseHM(h.HoraInicio)
		if err != nil {
			return nil, utils.BadRequest(err.Error())
		}
		fin, err := utils.ParseHM(h.HoraFin)
		if err != nil {
			return nil, utils.BadRequest(err.Error())
		}
		if fin <= ini {
			return nil, utils.BadRequest("la hora de fin debe ser posterior a la de inicio")
		}
		porDia[h.DiaSemana] = append(porDia[h.DiaSemana], [2]int{ini, fin})
		normalizados = append(normalizados, Horario{
			DiaSemana:  h.DiaSemana,
			HoraInicio: utils.FormatHM(ini),
			HoraFin:    utils.FormatHM(fin),
		})
	}

	for _, tramos := range porDia {
		if len(tramos) > maxTramosPorDia {
			return nil, utils.BadRequest("un día no puede tener más de 6 tramos de atención")
		}
		sort.Slice(tramos, func(i, j int) bool { return tramos[i][0] < tramos[j][0] })
		for i := 1; i < len(tramos); i++ {
			if tramos[i][0] < tramos[i-1][1] {
				return nil, utils.BadRequest("hay tramos de atención que se solapan en un mismo día")
			}
		}
	}
	return normalizados, nil
}

func (s *service) ReemplazarHorarios(ctx context.Context, id uuid.UUID, in HorariosInput) (*Espacio, error) {
	horarios, err := ValidarHorarios(in.Horarios)
	if err != nil {
		return nil, err
	}
	e, err := s.repo.Obtener(ctx, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, utils.NotFound("espacio no encontrado")
	}
	if err := s.repo.ReemplazarHorarios(ctx, id, horarios); err != nil {
		return nil, err
	}
	return s.repo.Obtener(ctx, id)
}
