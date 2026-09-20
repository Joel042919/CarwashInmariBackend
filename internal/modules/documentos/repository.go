package documentos

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	ReservaPerteneceACliente(ctx context.Context, idReserva, idCliente uuid.UUID) (bool, error)
	ReservaExiste(ctx context.Context, idReserva uuid.UUID) (bool, error)
	ServicioEnReserva(ctx context.Context, idReserva, idServicio uuid.UUID) (bool, error)
	ServicioRequiereDocumento(ctx context.Context, idServicio uuid.UUID) (existe bool, requiere bool, err error)
	EstadosExistentes(ctx context.Context, idReserva, idServicio uuid.UUID) ([]string, error)

	Crear(ctx context.Context, d *Documento) error
	ListarPorCliente(ctx context.Context, idCliente uuid.UUID, idReserva *uuid.UUID) ([]Documento, error)
	ListarTodos(ctx context.Context, estado string) ([]Documento, error)
	ObtenerPorID(ctx context.Context, id uuid.UUID) (*Documento, error)
	Resolver(ctx context.Context, id, idAdmin uuid.UUID, estado string, respuesta *string) (bool, error)
	Requisitos(ctx context.Context, idReserva uuid.UUID) ([]RequisitoDocumento, error)
}

type queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type repository struct {
	db queryer
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) existe(ctx context.Context, query string, args ...any) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&ok)
	return ok, err
}

func (r *repository) ReservaPerteneceACliente(ctx context.Context, idReserva, idCliente uuid.UUID) (bool, error) {
	return r.existe(ctx,
		`SELECT EXISTS(SELECT 1 FROM reservas WHERE id_reserva = $1 AND id_cliente = $2)`, idReserva, idCliente)
}

func (r *repository) ReservaExiste(ctx context.Context, idReserva uuid.UUID) (bool, error) {
	return r.existe(ctx, `SELECT EXISTS(SELECT 1 FROM reservas WHERE id_reserva = $1)`, idReserva)
}

func (r *repository) ServicioEnReserva(ctx context.Context, idReserva, idServicio uuid.UUID) (bool, error) {
	return r.existe(ctx,
		`SELECT EXISTS(SELECT 1 FROM reserva_servicios WHERE id_reserva = $1 AND id_servicio = $2)`,
		idReserva, idServicio)
}

func (r *repository) ServicioRequiereDocumento(ctx context.Context, idServicio uuid.UUID) (bool, bool, error) {
	var requiere bool
	err := r.db.QueryRowContext(ctx,
		`SELECT requiere_documento FROM servicios WHERE id_servicio = $1`, idServicio).Scan(&requiere)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, false, nil
		}
		return false, false, err
	}
	return true, requiere, nil
}

func (r *repository) EstadosExistentes(ctx context.Context, idReserva, idServicio uuid.UUID) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT estado::text FROM documentos_previos WHERE id_reserva = $1 AND id_servicio = $2`,
		idReserva, idServicio)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var estados []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		estados = append(estados, e)
	}
	return estados, rows.Err()
}

func (r *repository) Crear(ctx context.Context, d *Documento) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO documentos_previos (
			id_documento, id_reserva, id_servicio, id_cliente, ruta_pdf, estado, created_at
		) VALUES ($1, $2, $3, $4, $5, $6::estado_documento, $7)`,
		d.IDDocumento, d.IDReserva, d.IDServicio, d.IDCliente, d.RutaPDF, d.Estado, d.CreatedAt)
	return err
}

const selectDocumento = `
	SELECT d.id_documento, d.id_reserva, d.id_servicio, s.nombre, d.id_cliente,
		u.nombre || ' ' || u.apellido, u.correo, d.ruta_pdf, d.estado::text,
		d.admin_respuesta, d.validado_por, d.fecha_validacion, d.created_at
	FROM documentos_previos d
	INNER JOIN servicios s ON s.id_servicio = d.id_servicio
	INNER JOIN usuarios u ON u.id_usuario = d.id_cliente
`

type scanner interface {
	Scan(dest ...any) error
}

func scanDocumento(sc scanner, d *Documento) error {
	return sc.Scan(
		&d.IDDocumento, &d.IDReserva, &d.IDServicio, &d.NombreServicio, &d.IDCliente,
		&d.NombreCliente, &d.CorreoCliente, &d.RutaPDF, &d.Estado,
		&d.AdminRespuesta, &d.ValidadoPor, &d.FechaValidacion, &d.CreatedAt,
	)
}

func (r *repository) listar(ctx context.Context, query string, args ...any) ([]Documento, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []Documento{}
	for rows.Next() {
		var d Documento
		if err := scanDocumento(rows, &d); err != nil {
			return nil, err
		}
		lista = append(lista, d)
	}
	return lista, rows.Err()
}

func (r *repository) ListarPorCliente(ctx context.Context, idCliente uuid.UUID, idReserva *uuid.UUID) ([]Documento, error) {
	return r.listar(ctx, selectDocumento+`
		WHERE d.id_cliente = $1 AND ($2::uuid IS NULL OR d.id_reserva = $2::uuid)
		ORDER BY d.created_at DESC`, idCliente, idReserva)
}

func (r *repository) ListarTodos(ctx context.Context, estado string) ([]Documento, error) {
	return r.listar(ctx, selectDocumento+`
		WHERE ($1 = '' OR d.estado::text = $1)
		ORDER BY d.created_at DESC`, estado)
}

func (r *repository) ObtenerPorID(ctx context.Context, id uuid.UUID) (*Documento, error) {
	var d Documento
	err := scanDocumento(r.db.QueryRowContext(ctx, selectDocumento+` WHERE d.id_documento = $1`, id), &d)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// Resolver solo actúa sobre documentos en estado "adjuntado".
func (r *repository) Resolver(ctx context.Context, id, idAdmin uuid.UUID, estado string, respuesta *string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE documentos_previos
		SET estado = $1::estado_documento, admin_respuesta = $2, validado_por = $3, fecha_validacion = now()
		WHERE id_documento = $4 AND estado = 'adjuntado'`,
		estado, respuesta, idAdmin, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *repository) Requisitos(ctx context.Context, idReserva uuid.UUID) ([]RequisitoDocumento, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id_servicio, s.nombre, d.id_documento, d.estado::text
		FROM reserva_servicios rs
		INNER JOIN servicios s ON s.id_servicio = rs.id_servicio
		LEFT JOIN LATERAL (
			SELECT dp.id_documento, dp.estado
			FROM documentos_previos dp
			WHERE dp.id_reserva = rs.id_reserva AND dp.id_servicio = rs.id_servicio
			ORDER BY (dp.estado = 'validado') DESC, dp.created_at DESC
			LIMIT 1
		) d ON true
		WHERE rs.id_reserva = $1 AND s.requiere_documento
		ORDER BY s.nombre`, idReserva)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []RequisitoDocumento{}
	for rows.Next() {
		var rq RequisitoDocumento
		if err := rows.Scan(&rq.IDServicio, &rq.NombreServicio, &rq.IDDocumento, &rq.EstadoDocumento); err != nil {
			return nil, err
		}
		rq.Cumplido = rq.EstadoDocumento != nil && *rq.EstadoDocumento == EstadoValidado
		lista = append(lista, rq)
	}
	return lista, rows.Err()
}
