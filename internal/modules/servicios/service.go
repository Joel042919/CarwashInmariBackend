package servicios

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

type Service interface {
	ListarCategorias(ctx context.Context) ([]Categoria, error)
	CrearCategoria(ctx context.Context, in CategoriaInput) (*Categoria, error)
	ActualizarCategoria(ctx context.Context, id uuid.UUID, in CategoriaInput) (*Categoria, error)

	ListarServicios(ctx context.Context, soloDisponibles bool, idCategoria *uuid.UUID) ([]Servicio, error)
	ObtenerServicio(ctx context.Context, id uuid.UUID, soloDisponibles bool) (*Servicio, error)
	CrearServicio(ctx context.Context, in ServicioInput) (*Servicio, error)
	ActualizarServicio(ctx context.Context, id uuid.UUID, in ServicioInput) (*Servicio, error)
	CambiarDisponibilidad(ctx context.Context, id uuid.UUID, disponible bool) (*Servicio, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func validarCategoria(in *CategoriaInput) error {
	in.Nombre = strings.TrimSpace(in.Nombre)
	if in.Nombre == "" || len(in.Nombre) > 30 {
		return utils.BadRequest("el nombre de la categoría es obligatorio (máximo 30 caracteres)")
	}
	return nil
}

func (s *service) ListarCategorias(ctx context.Context) ([]Categoria, error) {
	return s.repo.ListarCategorias(ctx)
}

func (s *service) CrearCategoria(ctx context.Context, in CategoriaInput) (*Categoria, error) {
	if err := validarCategoria(&in); err != nil {
		return nil, err
	}
	c := &Categoria{IDCategoria: uuid.New(), Nombre: in.Nombre}
	if err := s.repo.CrearCategoria(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *service) ActualizarCategoria(ctx context.Context, id uuid.UUID, in CategoriaInput) (*Categoria, error) {
	if err := validarCategoria(&in); err != nil {
		return nil, err
	}
	c := &Categoria{IDCategoria: id, Nombre: in.Nombre}
	ok, err := s.repo.ActualizarCategoria(ctx, c)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, utils.NotFound("categoría no encontrada")
	}
	return c, nil
}

func (s *service) ListarServicios(ctx context.Context, soloDisponibles bool, idCategoria *uuid.UUID) ([]Servicio, error) {
	return s.repo.ListarServicios(ctx, soloDisponibles, idCategoria)
}

func (s *service) ObtenerServicio(ctx context.Context, id uuid.UUID, soloDisponibles bool) (*Servicio, error) {
	srv, err := s.repo.ObtenerServicio(ctx, id)
	if err != nil {
		return nil, err
	}
	if srv == nil || (soloDisponibles && !srv.Disponible) {
		return nil, utils.NotFound("servicio no encontrado")
	}
	return srv, nil
}

func (s *service) validarServicio(ctx context.Context, in *ServicioInput) error {
	in.Nombre = strings.TrimSpace(in.Nombre)
	if in.Nombre == "" || len(in.Nombre) > 80 {
		return utils.BadRequest("el nombre del servicio es obligatorio (máximo 80 caracteres)")
	}
	if in.Descripcion != nil {
		d := strings.TrimSpace(*in.Descripcion)
		if d == "" {
			in.Descripcion = nil
		} else {
			in.Descripcion = &d
		}
	}
	if in.Precio <= 0 {
		return utils.BadRequest("el precio debe ser mayor a 0")
	}
	if in.DuracionEstimadaMin <= 0 {
		return utils.BadRequest("la duración estimada debe ser mayor a 0 minutos")
	}
	if in.IDCategoria == uuid.Nil {
		return utils.BadRequest("la categoría es obligatoria")
	}
	existe, err := s.repo.ExisteCategoria(ctx, in.IDCategoria)
	if err != nil {
		return err
	}
	if !existe {
		return utils.BadRequest("la categoría indicada no existe")
	}
	return nil
}

func (s *service) CrearServicio(ctx context.Context, in ServicioInput) (*Servicio, error) {
	if err := s.validarServicio(ctx, &in); err != nil {
		return nil, err
	}
	disponible := true
	if in.Disponible != nil {
		disponible = *in.Disponible
	}
	srv := &Servicio{
		IDServicio:          uuid.New(),
		Nombre:              in.Nombre,
		Descripcion:         in.Descripcion,
		Precio:              in.Precio,
		DuracionEstimadaMin: in.DuracionEstimadaMin,
		RequiereDocumento:   in.RequiereDocumento,
		Disponible:          disponible,
		IDCategoria:         in.IDCategoria,
		CreatedAt:           time.Now(),
	}
	if err := s.repo.CrearServicio(ctx, srv); err != nil {
		return nil, err
	}
	return s.repo.ObtenerServicio(ctx, srv.IDServicio)
}

func (s *service) ActualizarServicio(ctx context.Context, id uuid.UUID, in ServicioInput) (*Servicio, error) {
	if err := s.validarServicio(ctx, &in); err != nil {
		return nil, err
	}
	actual, err := s.repo.ObtenerServicio(ctx, id)
	if err != nil {
		return nil, err
	}
	if actual == nil {
		return nil, utils.NotFound("servicio no encontrado")
	}
	disponible := actual.Disponible
	if in.Disponible != nil {
		disponible = *in.Disponible
	}
	srv := &Servicio{
		IDServicio:          id,
		Nombre:              in.Nombre,
		Descripcion:         in.Descripcion,
		Precio:              in.Precio,
		DuracionEstimadaMin: in.DuracionEstimadaMin,
		RequiereDocumento:   in.RequiereDocumento,
		Disponible:          disponible,
		IDCategoria:         in.IDCategoria,
	}
	if _, err := s.repo.ActualizarServicio(ctx, srv); err != nil {
		return nil, err
	}
	return s.repo.ObtenerServicio(ctx, id)
}

func (s *service) CambiarDisponibilidad(ctx context.Context, id uuid.UUID, disponible bool) (*Servicio, error) {
	ok, err := s.repo.CambiarDisponibilidad(ctx, id, disponible)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, utils.NotFound("servicio no encontrado")
	}
	return s.repo.ObtenerServicio(ctx, id)
}
