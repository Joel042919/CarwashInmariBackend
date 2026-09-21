package vehiculos

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Repository interface {
	Crear(ctx context.Context, v *Vehiculo) error
	ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]Vehiculo, error)
	Actualizar(ctx context.Context, v *Vehiculo) error
	Eliminar(ctx context.Context, idVehiculo, idCliente uuid.UUID) (bool, error)
}

func (r *repository) Actualizar(ctx context.Context, v *Vehiculo) error {
	result, err := r.db.ExecContext(ctx, `UPDATE vehiculos
		SET placa=$3, marca=$4, modelo=$5, color=$6, anio=$7, tipo_vehiculo=$8
		WHERE id_vehiculo=$1 AND id_cliente=$2`,
		v.IDVehiculo, v.IDCliente, v.Placa, v.Marca, v.Modelo, v.Color, v.Anio, v.TipoVehiculo)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *repository) Eliminar(ctx context.Context, idVehiculo, idCliente uuid.UUID) (bool, error) {
	// Un vehículo con reservas conserva su historial para no romper la trazabilidad.
	var usado bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM reservas WHERE id_vehiculo=$1)`, idVehiculo).Scan(&usado); err != nil {
		return false, err
	}
	if usado {
		return false, nil
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM vehiculos WHERE id_vehiculo=$1 AND id_cliente=$2`, idVehiculo, idCliente)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Crear(ctx context.Context, v *Vehiculo) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vehiculos (
			id_vehiculo, id_cliente, placa, marca, modelo, color, anio, tipo_vehiculo, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		v.IDVehiculo, v.IDCliente, v.Placa, v.Marca, v.Modelo, v.Color, v.Anio, v.TipoVehiculo, v.CreatedAt)
	return err
}

func (r *repository) ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]Vehiculo, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id_vehiculo, id_cliente, placa, marca, modelo, color, anio, tipo_vehiculo, created_at
		FROM vehiculos WHERE id_cliente = $1 ORDER BY created_at DESC`, idCliente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []Vehiculo{}
	for rows.Next() {
		var v Vehiculo
		if err := rows.Scan(&v.IDVehiculo, &v.IDCliente, &v.Placa, &v.Marca, &v.Modelo,
			&v.Color, &v.Anio, &v.TipoVehiculo, &v.CreatedAt); err != nil {
			return nil, err
		}
		lista = append(lista, v)
	}
	return lista, rows.Err()
}
