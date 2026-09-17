package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	BuscarPorCorreo(ctx context.Context, correo string) (*Usuario, error)
	BuscarRolPorNombre(ctx context.Context, nombreRol string) (uuid.UUID, error)
	BuscarPrimeraSede(ctx context.Context) (uuid.UUID, error)
	RegistrarClienteTx(ctx context.Context, u *Usuario, dni *string) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) BuscarPorCorreo(ctx context.Context, correo string) (*Usuario, error) {
	query := `
		SELECT 
			u.id_usuario, u.id_sede, u.nombre, u.apellido, 
			u.correo, u.telefono, u.contrasena_hash, u.id_rol, 
			r.rol, u.activo, u.fecha_registro
		FROM usuarios u
		INNER JOIN rol r ON u.id_rol = r.id
		WHERE u.correo = $1
	`
	row := r.db.QueryRowContext(ctx, query, correo)

	var u Usuario
	err := row.Scan(
		&u.IDUsuario,
		&u.IDSede,
		&u.Nombre,
		&u.Apellido,
		&u.Correo,
		&u.Telefono,
		&u.ContrasenaHash,
		&u.IDRol,
		&u.RolNombre,
		&u.Activo,
		&u.FechaRegistro,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

func (r *repository) BuscarRolPorNombre(ctx context.Context, nombreRol string) (uuid.UUID, error) {
	query := `SELECT id FROM rol WHERE LOWER(rol) = LOWER($1) LIMIT 1`
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, query, nombreRol).Scan(&id)
	return id, err
}

func (r *repository) BuscarPrimeraSede(ctx context.Context) (uuid.UUID, error) {
	query := `SELECT id FROM sede ORDER BY sede_numero ASC LIMIT 1`
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, query).Scan(&id)
	return id, err
}

// RegistrarClienteTx usa una transacción para crear el registro en "usuarios" y en "clientes"
func (r *repository) RegistrarClienteTx(ctx context.Context, u *Usuario, dni *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Insertar en tabla usuarios
	queryUsuario := `
		INSERT INTO usuarios (
			id_usuario, id_sede, nombre, apellido, correo, 
			telefono, contrasena_hash, id_rol, activo
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING fecha_registro
	`
	err = tx.QueryRowContext(ctx, queryUsuario,
		u.IDUsuario,
		u.IDSede,
		u.Nombre,
		u.Apellido,
		u.Correo,
		u.Telefono,
		u.ContrasenaHash,
		u.IDRol,
		u.Activo,
	).Scan(&u.FechaRegistro)

	if err != nil {
		return err
	}

	// 2. Insertar en tabla clientes (vinculado por id_usuario)
	queryCliente := `
		INSERT INTO clientes (id_usuario, dni)
		VALUES ($1, $2)
	`
	_, err = tx.ExecContext(ctx, queryCliente, u.IDUsuario, dni)
	if err != nil {
		return err
	}

	return tx.Commit()
}
