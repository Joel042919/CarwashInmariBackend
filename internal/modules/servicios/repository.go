package servicios

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	ListarCategorias(ctx context.Context) ([]Categoria, error)
	ExisteCategoria(ctx context.Context, id uuid.UUID) (bool, error)
	CrearCategoria(ctx context.Context, c *Categoria) error
	ActualizarCategoria(ctx context.Context, c *Categoria) (bool, error)

	ListarServicios(ctx context.Context, soloDisponibles bool, idCategoria *uuid.UUID) ([]Servicio, error)
	ObtenerServicio(ctx context.Context, id uuid.UUID) (*Servicio, error)
	CrearServicio(ctx context.Context, s *Servicio) error
	ActualizarServicio(ctx context.Context, s *Servicio) (bool, error)
	CambiarDisponibilidad(ctx context.Context, id uuid.UUID, disponible bool) (bool, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

const selectServicio = `
	SELECT s.id_servicio, s.nombre, s.descripcion, s.precio, s.duracion_estimada_min,
		s.requiere_documento, s.disponible, s.id_categoria, c.categoria_servicio, s.created_at
	FROM servicios s
	INNER JOIN categoria_servicios c ON c.id_categoria_servicio = s.id_categoria
`

type scanner interface {
	Scan(dest ...any) error
}

func scanServicio(sc scanner, s *Servicio) error {
	return sc.Scan(
		&s.IDServicio, &s.Nombre, &s.Descripcion, &s.Precio, &s.DuracionEstimadaMin,
		&s.RequiereDocumento, &s.Disponible, &s.IDCategoria, &s.Categoria, &s.CreatedAt,
	)
}

func (r *repository) ListarCategorias(ctx context.Context) ([]Categoria, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id_categoria_servicio, categoria_servicio FROM categoria_servicios ORDER BY categoria_servicio`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []Categoria{}
	for rows.Next() {
		var c Categoria
		if err := rows.Scan(&c.IDCategoria, &c.Nombre); err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, rows.Err()
}

func (r *repository) ExisteCategoria(ctx context.Context, id uuid.UUID) (bool, error) {
	var existe bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM categoria_servicios WHERE id_categoria_servicio = $1)`, id).Scan(&existe)
	return existe, err
}

func (r *repository) CrearCategoria(ctx context.Context, c *Categoria) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO categoria_servicios (id_categoria_servicio, categoria_servicio) VALUES ($1, $2)`,
		c.IDCategoria, c.Nombre)
	return err
}

func (r *repository) ActualizarCategoria(ctx context.Context, c *Categoria) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE categoria_servicios SET categoria_servicio = $1 WHERE id_categoria_servicio = $2`,
		c.Nombre, c.IDCategoria)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *repository) ListarServicios(ctx context.Context, soloDisponibles bool, idCategoria *uuid.UUID) ([]Servicio, error) {
	query := selectServicio + `
		WHERE ($1::boolean = false OR s.disponible)
		  AND ($2::uuid IS NULL OR s.id_categoria = $2::uuid)
		ORDER BY c.categoria_servicio, s.nombre
	`
	rows, err := r.db.QueryContext(ctx, query, soloDisponibles, idCategoria)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []Servicio{}
	for rows.Next() {
		var s Servicio
		if err := scanServicio(rows, &s); err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, rows.Err()
}

func (r *repository) ObtenerServicio(ctx context.Context, id uuid.UUID) (*Servicio, error) {
	var s Servicio
	err := scanServicio(r.db.QueryRowContext(ctx, selectServicio+` WHERE s.id_servicio = $1`, id), &s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *repository) CrearServicio(ctx context.Context, s *Servicio) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO servicios (
			id_servicio, nombre, descripcion, precio, duracion_estimada_min,
			requiere_documento, disponible, id_categoria, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		s.IDServicio, s.Nombre, s.Descripcion, s.Precio, s.DuracionEstimadaMin,
		s.RequiereDocumento, s.Disponible, s.IDCategoria, s.CreatedAt)
	return err
}

func (r *repository) ActualizarServicio(ctx context.Context, s *Servicio) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE servicios
		SET nombre = $1, descripcion = $2, precio = $3, duracion_estimada_min = $4,
			requiere_documento = $5, disponible = $6, id_categoria = $7
		WHERE id_servicio = $8`,
		s.Nombre, s.Descripcion, s.Precio, s.DuracionEstimadaMin,
		s.RequiereDocumento, s.Disponible, s.IDCategoria, s.IDServicio)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *repository) CambiarDisponibilidad(ctx context.Context, id uuid.UUID, disponible bool) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE servicios SET disponible = $1 WHERE id_servicio = $2`, disponible, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
