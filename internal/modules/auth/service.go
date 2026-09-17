package auth

import (
	"context"
	"errors"
	"strings"

	"carwashinmaribackend/internal/utils"

	"github.com/google/uuid"
)

type Service interface {
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	RegistrarCliente(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	req.Correo = strings.TrimSpace(strings.ToLower(req.Correo))
	if req.Correo == "" || req.Contrasena == "" {
		return nil, errors.New("correo y contraseña son obligatorios")
	}

	usuario, err := s.repo.BuscarPorCorreo(ctx, req.Correo)
	if err != nil {
		return nil, err
	}
	if usuario == nil {
		return nil, errors.New("credenciales inválidas")
	}

	if !usuario.Activo {
		return nil, errors.New("el usuario se encuentra inactivo, contacte al administrador")
	}

	if !utils.CheckPasswordHash(req.Contrasena, usuario.ContrasenaHash) {
		return nil, errors.New("credenciales inválidas")
	}

	token, err := utils.GenerarJWT(usuario.IDUsuario, usuario.IDSede, usuario.RolNombre)
	if err != nil {
		return nil, errors.New("error al generar token de acceso")
	}

	return &AuthResponse{
		Token:   token,
		Usuario: *usuario,
	}, nil
}

func (s *service) RegistrarCliente(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	req.Correo = strings.TrimSpace(strings.ToLower(req.Correo))
	req.Nombre = strings.TrimSpace(req.Nombre)
	req.Apellido = strings.TrimSpace(req.Apellido)

	if req.Correo == "" || req.Contrasena == "" || req.Nombre == "" || req.Apellido == "" {
		return nil, errors.New("todos los campos obligatorios deben ser completados")
	}

	if len(req.Contrasena) < 6 {
		return nil, errors.New("la contraseña debe tener al menos 6 caracteres")
	}

	existente, err := s.repo.BuscarPorCorreo(ctx, req.Correo)
	if err != nil {
		return nil, err
	}
	if existente != nil {
		return nil, errors.New("el correo electrónico ya se encuentra registrado")
	}

	// Si no se envió IDSede, asignar la primera sede disponible
	sedeID := req.IDSede
	if sedeID == uuid.Nil {
		primeraSede, err := s.repo.BuscarPrimeraSede(ctx)
		if err != nil {
			return nil, errors.New("no hay sedes activas registradas en el sistema")
		}
		sedeID = primeraSede
	}

	// Obtener ID del rol 'cliente'
	rolClienteID, err := s.repo.BuscarRolPorNombre(ctx, "cliente")
	if err != nil {
		return nil, errors.New("error al obtener rol de cliente predeterminado")
	}

	hash, err := utils.HashPassword(req.Contrasena)
	if err != nil {
		return nil, errors.New("error al procesar contraseña")
	}

	nuevoUsuario := &Usuario{
		IDUsuario:      uuid.New(),
		IDSede:         sedeID,
		Nombre:         req.Nombre,
		Apellido:       req.Apellido,
		Correo:         req.Correo,
		Telefono:       req.Telefono,
		ContrasenaHash: hash,
		IDRol:          rolClienteID,
		RolNombre:      "cliente",
		Activo:         true,
	}

	if err := s.repo.RegistrarClienteTx(ctx, nuevoUsuario, req.DNI); err != nil {
		return nil, errors.New("error al registrar cliente en el sistema: " + err.Error())
	}

	token, err := utils.GenerarJWT(nuevoUsuario.IDUsuario, nuevoUsuario.IDSede, nuevoUsuario.RolNombre)
	if err != nil {
		return nil, errors.New("registro exitoso pero falló la generación de sesión")
	}

	return &AuthResponse{
		Token:   token,
		Usuario: *nuevoUsuario,
	}, nil
}
