package trabajadores

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

var (
	correoValido = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	dniValido    = regexp.MustCompile(`^[0-9A-Za-z]{6,20}$`)
)

type Service interface {
	Crear(ctx context.Context, idSede uuid.UUID, in CrearTrabajadorInput) (*Trabajador, error)
	Listar(ctx context.Context) ([]Trabajador, error)
	CambiarDisponibilidad(ctx context.Context, id uuid.UUID, disponible bool) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Crear(ctx context.Context, idSede uuid.UUID, in CrearTrabajadorInput) (*Trabajador, error) {
	nombre := strings.TrimSpace(in.Nombre)
	apellido := strings.TrimSpace(in.Apellido)
	correo := strings.ToLower(strings.TrimSpace(in.Correo))
	dni := strings.TrimSpace(in.DNI)

	if nombre == "" || apellido == "" || len(nombre) > 100 || len(apellido) > 100 {
		return nil, utils.BadRequest("el nombre y el apellido son obligatorios")
	}
	if !correoValido.MatchString(correo) || len(correo) > 150 {
		return nil, utils.BadRequest("el correo electrónico no es válido")
	}
	if !dniValido.MatchString(dni) {
		return nil, utils.BadRequest("el documento de identidad debe tener entre 6 y 20 caracteres")
	}
	if len(in.Contrasena) < 6 {
		return nil, utils.BadRequest("la contraseña debe tener al menos 6 caracteres")
	}

	fecha := strings.TrimSpace(in.FechaContratacion)
	if fecha == "" {
		fecha = time.Now().Format("2006-01-02")
	} else if _, err := utils.ParseFecha(fecha); err != nil {
		return nil, utils.BadRequest(err.Error())
	}

	var telefono *string
	if in.Telefono != nil {
		if t := strings.TrimSpace(*in.Telefono); t != "" {
			if len(t) > 20 {
				return nil, utils.BadRequest("el teléfono no puede superar 20 caracteres")
			}
			telefono = &t
		}
	}

	hash, err := utils.HashPassword(in.Contrasena)
	if err != nil {
		return nil, err
	}

	t := &Trabajador{
		IDUsuario:         uuid.New(),
		Nombre:            nombre,
		Apellido:          apellido,
		Correo:            correo,
		Telefono:          telefono,
		DNI:               dni,
		FechaContratacion: fecha,
		Disponible:        true,
	}
	if err := s.repo.Crear(ctx, idSede, t, hash); err != nil {
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe un usuario con ese correo o documento de identidad")
		}
		return nil, err
	}
	return t, nil
}

func (s *service) Listar(ctx context.Context) ([]Trabajador, error) {
	return s.repo.Listar(ctx)
}

func (s *service) CambiarDisponibilidad(ctx context.Context, id uuid.UUID, disponible bool) error {
	ok, err := s.repo.CambiarDisponibilidad(ctx, id, disponible)
	if err != nil {
		return err
	}
	if !ok {
		return utils.NotFound("trabajador no encontrado")
	}
	return nil
}
