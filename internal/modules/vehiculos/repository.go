package vehiculos

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Repository interface {
	Crear(ctx context.Context, v *Vehiculo) error
	ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]Vehiculo, error)
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
