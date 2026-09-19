package espacios

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	Listar(ctx context.Context, soloActivos bool) ([]Espacio, error)
	Obtener(ctx context.Context, id uuid.UUID) (*Espacio, error)
	Crear(ctx context.Context, e *Espacio) error
	Actualizar(ctx context.Context, e *Espacio) (bool, error)
	ReemplazarHorarios(ctx context.Context, id uuid.UUID, horarios []Horario) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) cargarHorarios(ctx context.Context, espacios []Espacio) error {
	for i := range espacios {
		rows, err := r.db.QueryContext(ctx, `
			SELECT dia_semana, to_char(hora_inicio, 'HH24:MI'), to_char(hora_fin, 'HH24:MI')
			FROM horarios_atencion
			WHERE id_espacio = $1
			ORDER BY dia_semana, hora_inicio`, espacios[i].IDEspacio)
		if err != nil {
			return err
		}
		espacios[i].Horarios = []Horario{}
		for rows.Next() {
			var h Horario
			if err := rows.Scan(&h.DiaSemana, &h.HoraInicio, &h.HoraFin); err != nil {
				rows.Close()
				return err
			}
			espacios[i].Horarios = append(espacios[i].Horarios, h)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
	}
	return nil
}

func (r *repository) Listar(ctx context.Context, soloActivos bool) ([]Espacio, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id_espacio, codigo, activo FROM espacios_lavado
		WHERE ($1::boolean = false OR activo)
		ORDER BY codigo`, soloActivos)
	if err != nil {
		return nil, err
	}
	lista := []Espacio{}
	for rows.Next() {
		var e Espacio
		if err := rows.Scan(&e.IDEspacio, &e.Codigo, &e.Activo); err != nil {
			rows.Close()
			return nil, err
		}
		lista = append(lista, e)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if err := r.cargarHorarios(ctx, lista); err != nil {
		return nil, err
	}
	return lista, nil
}

func (r *repository) Obtener(ctx context.Context, id uuid.UUID) (*Espacio, error) {
	var e Espacio
	err := r.db.QueryRowContext(ctx,
		`SELECT id_espacio, codigo, activo FROM espacios_lavado WHERE id_espacio = $1`, id).
		Scan(&e.IDEspacio, &e.Codigo, &e.Activo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	lista := []Espacio{e}
	if err := r.cargarHorarios(ctx, lista); err != nil {
		return nil, err
	}
	return &lista[0], nil
}

func (r *repository) Crear(ctx context.Context, e *Espacio) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO espacios_lavado (id_espacio, codigo, activo) VALUES ($1, $2, $3)`,
		e.IDEspacio, e.Codigo, e.Activo)
	return err
}

func (r *repository) Actualizar(ctx context.Context, e *Espacio) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE espacios_lavado SET codigo = $1, activo = $2 WHERE id_espacio = $3`,
		e.Codigo, e.Activo, e.IDEspacio)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ReemplazarHorarios sustituye todo el horario semanal del espacio en una transacción.
func (r *repository) ReemplazarHorarios(ctx context.Context, id uuid.UUID, horarios []Horario) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM horarios_atencion WHERE id_espacio = $1`, id); err != nil {
		return err
	}
	for _, h := range horarios {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO horarios_atencion (id_horario, id_espacio, dia_semana, hora_inicio, hora_fin)
			VALUES ($1, $2, $3, $4::time, $5::time)`,
			uuid.New(), id, h.DiaSemana, h.HoraInicio, h.HoraFin); err != nil {
			return err
		}
	}
	return tx.Commit()
}
