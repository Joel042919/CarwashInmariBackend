package vehiculos

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

var placaValida = regexp.MustCompile(`^[A-Z0-9-]{5,15}$`)

type Service interface {
	Registrar(ctx context.Context, idCliente uuid.UUID, in VehiculoInput) (*Vehiculo, error)
	MisVehiculos(ctx context.Context, idCliente uuid.UUID) ([]Vehiculo, error)
	Actualizar(ctx context.Context, idCliente, idVehiculo uuid.UUID, in VehiculoInput) (*Vehiculo, error)
	Eliminar(ctx context.Context, idCliente, idVehiculo uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func limpiar(s *string, max int) (*string, error) {
	if s == nil {
		return nil, nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil, nil
	}
	if len(t) > max {
		return nil, utils.BadRequest("un campo del vehículo excede la longitud permitida")
	}
	return &t, nil
}

func (s *service) construir(idCliente uuid.UUID, in VehiculoInput) (*Vehiculo, error) {
	placa := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(in.Placa), " ", ""))
	if !placaValida.MatchString(placa) {
		return nil, utils.BadRequest("la placa debe tener entre 5 y 15 letras, números o guiones")
	}
	marca := strings.TrimSpace(in.Marca)
	modelo := strings.TrimSpace(in.Modelo)
	if marca == "" || len(marca) > 50 || modelo == "" || len(modelo) > 50 {
		return nil, utils.BadRequest("la marca y el modelo son obligatorios (máximo 50 caracteres)")
	}
	color, err := limpiar(in.Color, 30)
	if err != nil {
		return nil, err
	}
	tipo, err := limpiar(in.TipoVehiculo, 30)
	if err != nil {
		return nil, err
	}
	if in.Anio != nil && (*in.Anio < 1950 || *in.Anio > time.Now().Year()+1) {
		return nil, utils.BadRequest("el año del vehículo no es válido")
	}

	return &Vehiculo{
		IDCliente:    idCliente,
		Placa:        placa,
		Marca:        marca,
		Modelo:       modelo,
		Color:        color,
		Anio:         in.Anio,
		TipoVehiculo: tipo,
		CreatedAt:    time.Now(),
	}, nil
}

func (s *service) Registrar(ctx context.Context, idCliente uuid.UUID, in VehiculoInput) (*Vehiculo, error) {
	v, err := s.construir(idCliente, in)
	if err != nil {
		return nil, err
	}
	v.IDVehiculo = uuid.New()
	v.CreatedAt = time.Now()
	if err := s.repo.Crear(ctx, v); err != nil {
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe un vehículo registrado con esa placa")
		}
		return nil, err
	}
	return v, nil
}

func (s *service) MisVehiculos(ctx context.Context, idCliente uuid.UUID) ([]Vehiculo, error) {
	return s.repo.ListarPorCliente(ctx, idCliente)
}

func (s *service) Actualizar(ctx context.Context, idCliente, idVehiculo uuid.UUID, in VehiculoInput) (*Vehiculo, error) {
	v, err := s.construir(idCliente, in)
	if err != nil {
		return nil, err
	}
	v.IDVehiculo = idVehiculo
	if err = s.repo.Actualizar(ctx, v); err != nil {
		if err == sql.ErrNoRows {
			return nil, utils.NotFound("vehículo no encontrado")
		}
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe un vehículo registrado con esa placa")
		}
		return nil, err
	}
	return v, nil
}

func (s *service) Eliminar(ctx context.Context, idCliente, idVehiculo uuid.UUID) error {
	eliminado, err := s.repo.Eliminar(ctx, idVehiculo, idCliente)
	if err != nil {
		return err
	}
	if !eliminado {
		return utils.Conflict("no se puede eliminar un vehículo que ya tiene reservas, o no existe")
	}
	return nil
}
