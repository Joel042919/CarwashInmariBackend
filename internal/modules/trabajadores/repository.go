package trabajadores

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	Crear(ctx context.Context, idSede uuid.UUID, t *Trabajador, hash string) error
	Listar(ctx context.Context) ([]Trabajador, error)
	CambiarDisponibilidad(ctx context.Context, id uuid.UUID, disponible bool) (bool, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// rolTrabajador devuelve el id del rol "trabajador"; si aún no existe lo crea.
func rolTrabajador(ctx context.Context, tx *sql.Tx) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRowContext(ctx, `SELECT id FROM rol WHERE LOWER(rol) = 'trabajador' LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, err
	}
	id = uuid.New()
	if _, err := tx.ExecContext(ctx, `INSERT INTO rol (id, rol) VALUES ($1, 'trabajador')`, id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// Crear registra al trabajador en "usuarios" (rol trabajador) y en "trabajadores".
func (r *repository) Crear(ctx context.Context, idSede uuid.UUID, t *Trabajador, hash string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	idRol, err := rolTrabajador(ctx, tx)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO usuarios (id_usuario, id_sede, nombre, apellido, correo, telefono, contrasena_hash, id_rol, activo)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)`,
		t.IDUsuario, idSede, t.Nombre, t.Apellido, t.Correo, t.Telefono, hash, idRol); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO trabajadores (id_usuario, dni, fecha_contratacion, disponible)
		VALUES ($1, $2, $3::date, true)`,
		t.IDUsuario, t.DNI, t.FechaContratacion); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *repository) Listar(ctx context.Context) ([]Trabajador, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id_usuario, u.nombre, u.apellido, u.correo, u.telefono,
			t.dni, t.fecha_contratacion::text, t.disponible
		FROM trabajadores t
		INNER JOIN usuarios u ON u.id_usuario = t.id_usuario
		WHERE t.fecha_cese IS NULL
		ORDER BY u.nombre, u.apellido`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []Trabajador{}
	for rows.Next() {
		var t Trabajador
		if err := rows.Scan(&t.IDUsuario, &t.Nombre, &t.Apellido, &t.Correo, &t.Telefono,
			&t.DNI, &t.FechaContratacion, &t.Disponible); err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}
	return lista, rows.Err()
}

func (r *repository) CambiarDisponibilidad(ctx context.Context, id uuid.UUID, disponible bool) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE trabajadores SET disponible = $1 WHERE id_usuario = $2`, disponible, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
