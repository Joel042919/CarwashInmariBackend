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
	Listar(ctx context.Context, sede uuid.UUID) ([]Trabajador, error)
	CambiarDisponibilidad(ctx context.Context, sede, id uuid.UUID, disponible bool) error
	Actualizar(ctx context.Context, sede, id uuid.UUID, in ActualizarTrabajadorInput) (*Trabajador, error)
	DarBaja(ctx context.Context, sede, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Crear(ctx context.Context, idSede uuid.UUID, in CrearTrabajadorInput) (*Trabajador, error) {
	if len(in.Contrasena) < 6 || len(in.Contrasena) > 72 {
		return nil, utils.BadRequest("la contrase?a debe tener entre 6 y 72 bytes")
	}
	if strings.TrimSpace(in.FechaContratacion) == "" {
		in.FechaContratacion = time.Now().In(utils.ZonaNegocio).Format("2006-01-02")
	}
	datos, err := validarDatos(ActualizarTrabajadorInput{Nombre: in.Nombre, Apellido: in.Apellido, Correo: in.Correo, Telefono: in.Telefono, DNI: in.DNI, FechaContratacion: in.FechaContratacion})
	if err != nil {
		return nil, err
	}

	hash, err := utils.HashPassword(in.Contrasena)
	if err != nil {
		return nil, err
	}

	t := &Trabajador{
		IDUsuario:         uuid.New(),
		Nombre:            datos.Nombre,
		Apellido:          datos.Apellido,
		Correo:            datos.Correo,
		Telefono:          datos.Telefono,
		DNI:               datos.DNI,
		FechaContratacion: datos.FechaContratacion,
		Disponible:        true,
		Activo:            true,
	}
	if err := s.repo.Crear(ctx, idSede, t, hash); err != nil {
		if utils.IsUniqueViolation(err) {
			return nil, utils.Conflict("ya existe un usuario con ese correo o documento de identidad")
		}
		return nil, err
	}
	return t, nil
}

func (s *service) Listar(ctx context.Context, sede uuid.UUID) ([]Trabajador, error) {
	return s.repo.Listar(ctx, sede)
}

func (s *service) CambiarDisponibilidad(ctx context.Context, sede, id uuid.UUID, disponible bool) error {
	ok, err := s.repo.CambiarDisponibilidad(ctx, sede, id, disponible)
	if err != nil {
		return err
	}
	if !ok {
		return utils.NotFound("trabajador no encontrado")
	}
	return nil
}

func (s *service) Actualizar(ctx context.Context, sede, id uuid.UUID, in ActualizarTrabajadorInput) (*Trabajador, error) {
	in, err := validarDatos(in)
	if err != nil {
		return nil, err
	}
	t, err := s.repo.Actualizar(ctx, sede, id, in)
	if utils.IsUniqueViolation(err) {
		return nil, utils.Conflict("ya existe un usuario con ese correo o documento de identidad")
	}
	return t, err
}

func (s *service) DarBaja(ctx context.Context, sede, id uuid.UUID) error {
	return s.repo.DarBaja(ctx, sede, id)
}

func validarDatos(in ActualizarTrabajadorInput) (ActualizarTrabajadorInput, error) {
	in.Nombre = strings.TrimSpace(in.Nombre)
	in.Apellido = strings.TrimSpace(in.Apellido)
	in.Correo = strings.ToLower(strings.TrimSpace(in.Correo))
	in.FechaContratacion = strings.TrimSpace(in.FechaContratacion)
	in.DNI = strings.TrimSpace(in.DNI)
	if in.Nombre == "" || in.Apellido == "" || len(in.Nombre) > 100 || len(in.Apellido) > 100 {
		return in, utils.BadRequest("nombre y apellido son obligatorios y admiten hasta 100 caracteres")
	}
	if !correoValido.MatchString(in.Correo) || len(in.Correo) > 150 {
		return in, utils.BadRequest("el correo electrónico no es válido")
	}
	if !dniValido.MatchString(in.DNI) {
		return in, utils.BadRequest("documento de identidad inválido")
	}
	if _, err := utils.ParseRangoFechas(in.FechaContratacion, ""); err != nil || in.FechaContratacion == "" {
		return in, utils.BadRequest("fecha de contratación inválida")
	}
	if in.Telefono != nil {
		telefono := strings.TrimSpace(*in.Telefono)
		if len(telefono) > 20 {
			return in, utils.BadRequest("el teléfono admite hasta 20 caracteres")
		}
		in.Telefono = nil
		if telefono != "" {
			in.Telefono = &telefono
		}
	}
	return in, nil
}
