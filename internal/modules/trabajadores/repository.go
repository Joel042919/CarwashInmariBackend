package trabajadores

import (
	"carwashinmaribackend/internal/utils"
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	Crear(ctx context.Context, idSede uuid.UUID, t *Trabajador, hash string) error
	Listar(ctx context.Context, sede uuid.UUID) ([]Trabajador, error)
	CambiarDisponibilidad(ctx context.Context, sede, id uuid.UUID, disponible bool) (bool, error)
	Actualizar(ctx context.Context, sede, id uuid.UUID, in ActualizarTrabajadorInput) (*Trabajador, error)
	DarBaja(ctx context.Context, sede, id uuid.UUID) error
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

func (r *repository) Listar(ctx context.Context, sede uuid.UUID) ([]Trabajador, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id_usuario, u.nombre, u.apellido, u.correo, u.telefono,
			t.dni, t.fecha_contratacion::text, t.disponible, u.activo, t.fecha_cese::text
		FROM trabajadores t
		INNER JOIN usuarios u ON u.id_usuario = t.id_usuario
		WHERE u.id_sede = $1
		ORDER BY u.activo DESC, u.nombre, u.apellido`, sede)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []Trabajador{}
	for rows.Next() {
		var t Trabajador
		if err := rows.Scan(&t.IDUsuario, &t.Nombre, &t.Apellido, &t.Correo, &t.Telefono,
			&t.DNI, &t.FechaContratacion, &t.Disponible, &t.Activo, &t.FechaCese); err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}
	return lista, rows.Err()
}

func (r *repository) CambiarDisponibilidad(ctx context.Context, sede, id uuid.UUID, disponible bool) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE trabajadores t SET disponible = $1 FROM usuarios u
         WHERE t.id_usuario = $2 AND u.id_usuario = t.id_usuario AND u.id_sede = $3
         AND u.activo AND t.fecha_cese IS NULL`, disponible, id, sede)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// Bloquear la fila laboral serializa bajas, edición y asignación de personal.
func bloquearTrabajador(ctx context.Context, tx *sql.Tx, sede, id uuid.UUID) error {
	var activo bool
	err := tx.QueryRowContext(ctx, `SELECT u.activo AND t.fecha_cese IS NULL
 FROM trabajadores t JOIN usuarios u ON u.id_usuario=t.id_usuario
 WHERE t.id_usuario=$1 AND u.id_sede=$2 FOR UPDATE OF t, u`, id, sede).Scan(&activo)
	if errors.Is(err, sql.ErrNoRows) {
		return utils.NotFound("trabajador no encontrado")
	}
	if err != nil {
		return err
	}
	if !activo {
		return utils.Conflict("el trabajador ya está dado de baja")
	}
	return nil
}

func (r *repository) Actualizar(ctx context.Context, sede, id uuid.UUID, in ActualizarTrabajadorInput) (*Trabajador, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = bloquearTrabajador(ctx, tx, sede, id); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE usuarios SET nombre=$2, apellido=$3, correo=$4, telefono=$5 WHERE id_usuario=$1`, id, in.Nombre, in.Apellido, in.Correo, in.Telefono); err != nil {
		return nil, err
	}
	t := &Trabajador{IDUsuario: id, Nombre: in.Nombre, Apellido: in.Apellido, Correo: in.Correo, Telefono: in.Telefono, DNI: in.DNI, FechaContratacion: in.FechaContratacion, Activo: true}
	err = tx.QueryRowContext(ctx, `UPDATE trabajadores SET dni=$2, fecha_contratacion=$3::date WHERE id_usuario=$1 RETURNING disponible`, id, in.DNI, in.FechaContratacion).Scan(&t.Disponible)
	if err != nil {
		return nil, err
	}
	return t, tx.Commit()
}

func (r *repository) DarBaja(ctx context.Context, sede, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = bloquearTrabajador(ctx, tx, sede, id); err != nil {
		return err
	}
	var pendiente bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM asignaciones_trabajadores x
 JOIN atenciones a ON a.id_atencion=x.id_atencion JOIN reservas r ON r.id_reserva=a.id_reserva
 WHERE x.id_trabajador=$1 AND r.estado <> 'cancelada' AND a.estado IN ('programada','en_proceso','en_pausa'))`, id).Scan(&pendiente)
	if err != nil {
		return err
	}
	if pendiente {
		return utils.Conflict("tiene atenciones pendientes: consulta sus asignaciones y reasigna las programadas o finaliza las que están en curso antes de dar la baja")
	}
	if _, err = tx.ExecContext(ctx, `UPDATE trabajadores SET disponible=false,fecha_cese=(now() AT TIME ZONE 'America/Lima')::date WHERE id_usuario=$1`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE usuarios SET activo=false WHERE id_usuario=$1`, id); err != nil {
		return err
	}
	return tx.Commit()
}
