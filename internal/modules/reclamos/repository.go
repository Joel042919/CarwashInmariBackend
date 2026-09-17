package reclamos

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CrearReclamoConEvidenciasTx(ctx context.Context, r *Reclamo) error
	ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]Reclamo, error)
	ListarTodos(ctx context.Context) ([]Reclamo, error)
	ObtenerPorID(ctx context.Context, idReclamo uuid.UUID) (*Reclamo, error)
	ResponderReclamo(ctx context.Context, idReclamo, idAdmin uuid.UUID, respuesta string, estado EstadoReclamo) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CrearReclamoConEvidenciasTx(ctx context.Context, rec *Reclamo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queryReclamo := `
		INSERT INTO reclamos (
			id_reclamo, id_cliente, id_atencion, id_pago, id_pedido,
			asunto, descripcion, estado, fecha_registro
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.ExecContext(ctx, queryReclamo,
		rec.IDReclamo,
		rec.IDCliente,
		rec.IDAtencion,
		rec.IDPago,
		rec.IDPedido,
		rec.Asunto,
		rec.Descripcion,
		rec.Estado,
		rec.FechaRegistro,
	)
	if err != nil {
		return err
	}

	queryEvidencia := `
		INSERT INTO evidencias_reclamo (
			id_evidencia, id_reclamo, ruta_archivo, descripcion, created_at
		) VALUES ($1, $2, $3, $4, $5)
	`
	for _, ev := range rec.Evidencias {
		_, err = tx.ExecContext(ctx, queryEvidencia,
			ev.IDEvidencia,
			rec.IDReclamo,
			ev.RutaArchivo,
			ev.Descripcion,
			ev.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *repository) ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]Reclamo, error) {
	query := `
		SELECT 
			id_reclamo, id_cliente, id_atencion, id_pago, id_pedido,
			asunto, descripcion, estado, respuesta_admin, respondido_por,
			fecha_registro, fecha_respuesta
		FROM reclamos
		WHERE id_cliente = $1
		ORDER BY fecha_registro DESC
	`
	rows, err := r.db.QueryContext(ctx, query, idCliente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []Reclamo
	for rows.Next() {
		var rec Reclamo
		if err := rows.Scan(
			&rec.IDReclamo,
			&rec.IDCliente,
			&rec.IDAtencion,
			&rec.IDPago,
			&rec.IDPedido,
			&rec.Asunto,
			&rec.Descripcion,
			&rec.Estado,
			&rec.RespuestaAdmin,
			&rec.RespondidoPor,
			&rec.FechaRegistro,
			&rec.FechaRespuesta,
		); err != nil {
			return nil, err
		}
		evs, _ := r.obtenerEvidenciasPorReclamo(ctx, rec.IDReclamo)
		if evs == nil {
			evs = []EvidenciaReclamo{}
		}
		rec.Evidencias = evs
		lista = append(lista, rec)
	}

	if lista == nil {
		lista = []Reclamo{}
	}
	return lista, nil
}

func (r *repository) ListarTodos(ctx context.Context) ([]Reclamo, error) {
	query := `
		SELECT 
			r.id_reclamo, r.id_cliente, u.nombre || ' ' || u.apellido as nombre_cliente, u.correo,
			r.id_atencion, r.id_pago, r.id_pedido, r.asunto, r.descripcion, 
			r.estado, r.respuesta_admin, r.respondido_por, r.fecha_registro, r.fecha_respuesta
		FROM reclamos r
		INNER JOIN usuarios u ON r.id_cliente = u.id_usuario
		ORDER BY r.fecha_registro DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []Reclamo
	for rows.Next() {
		var rec Reclamo
		if err := rows.Scan(
			&rec.IDReclamo,
			&rec.IDCliente,
			&rec.NombreCliente,
			&rec.CorreoCliente,
			&rec.IDAtencion,
			&rec.IDPago,
			&rec.IDPedido,
			&rec.Asunto,
			&rec.Descripcion,
			&rec.Estado,
			&rec.RespuestaAdmin,
			&rec.RespondidoPor,
			&rec.FechaRegistro,
			&rec.FechaRespuesta,
		); err != nil {
			return nil, err
		}
		evs, _ := r.obtenerEvidenciasPorReclamo(ctx, rec.IDReclamo)
		if evs == nil {
			evs = []EvidenciaReclamo{}
		}
		rec.Evidencias = evs
		lista = append(lista, rec)
	}

	if lista == nil {
		lista = []Reclamo{}
	}
	return lista, nil
}

func (r *repository) ObtenerPorID(ctx context.Context, idReclamo uuid.UUID) (*Reclamo, error) {
	query := `
		SELECT 
			id_reclamo, id_cliente, id_atencion, id_pago, id_pedido,
			asunto, descripcion, estado, respuesta_admin, respondido_por,
			fecha_registro, fecha_respuesta
		FROM reclamos
		WHERE id_reclamo = $1
	`
	row := r.db.QueryRowContext(ctx, query, idReclamo)

	var rec Reclamo
	err := row.Scan(
		&rec.IDReclamo,
		&rec.IDCliente,
		&rec.IDAtencion,
		&rec.IDPago,
		&rec.IDPedido,
		&rec.Asunto,
		&rec.Descripcion,
		&rec.Estado,
		&rec.RespuestaAdmin,
		&rec.RespondidoPor,
		&rec.FechaRegistro,
		&rec.FechaRespuesta,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	evs, err := r.obtenerEvidenciasPorReclamo(ctx, rec.IDReclamo)
	if err != nil {
		return nil, err
	}
	if evs == nil {
		evs = []EvidenciaReclamo{}
	}
	rec.Evidencias = evs

	return &rec, nil
}

func (r *repository) ResponderReclamo(ctx context.Context, idReclamo, idAdmin uuid.UUID, respuesta string, estado EstadoReclamo) error {
	query := `
		UPDATE reclamos
		SET respuesta_admin = $1, estado = $2, respondido_por = $3, fecha_respuesta = $4
		WHERE id_reclamo = $5
	`
	now := time.Now()
	res, err := r.db.ExecContext(ctx, query, respuesta, estado, idAdmin, now, idReclamo)
	if err != nil {
		return err
	}

	filas, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if filas == 0 {
		return errors.New("reclamo no encontrado")
	}

	return nil
}

func (r *repository) obtenerEvidenciasPorReclamo(ctx context.Context, idReclamo uuid.UUID) ([]EvidenciaReclamo, error) {
	query := `
		SELECT id_evidencia, id_reclamo, ruta_archivo, descripcion, created_at
		FROM evidencias_reclamo
		WHERE id_reclamo = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, idReclamo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []EvidenciaReclamo
	for rows.Next() {
		var ev EvidenciaReclamo
		if err := rows.Scan(
			&ev.IDEvidencia,
			&ev.IDReclamo,
			&ev.RutaArchivo,
			&ev.Descripcion,
			&ev.CreatedAt,
		); err != nil {
			return nil, err
		}
		lista = append(lista, ev)
	}
	return lista, nil
}
