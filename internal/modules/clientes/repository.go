package clientes

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Repository interface {
	ObtenerPerfil(ctx context.Context, idUsuario uuid.UUID) (*ClientePerfil, error)
	ActualizarPerfil(ctx context.Context, idUsuario uuid.UUID, req UpdateClienteRequest) error
	ListarReservasPorCliente(ctx context.Context, idCliente uuid.UUID) ([]ReservaHistorial, error)
	ListarAtencionesPorCliente(ctx context.Context, idCliente uuid.UUID) ([]AtencionHistorial, error)
	ListarPagosPorCliente(ctx context.Context, idCliente uuid.UUID) ([]PagoHistorial, error)
	ListarPedidosPorCliente(ctx context.Context, idCliente uuid.UUID) ([]PedidoHistorial, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) ObtenerPerfil(ctx context.Context, idUsuario uuid.UUID) (*ClientePerfil, error) {
	query := `
		SELECT 
			u.id_usuario, u.id_sede, u.nombre, u.apellido, u.correo, u.telefono,
			c.dni, c.fecha_nacimiento, c.direccion, c.notas, COALESCE(c.created_at, u.fecha_registro)
		FROM usuarios u
		LEFT JOIN clientes c ON u.id_usuario = c.id_usuario
		WHERE u.id_usuario = $1
	`
	row := r.db.QueryRowContext(ctx, query, idUsuario)

	var p ClientePerfil
	err := row.Scan(
		&p.IDUsuario,
		&p.IDSede,
		&p.Nombre,
		&p.Apellido,
		&p.Correo,
		&p.Telefono,
		&p.DNI,
		&p.FechaNacimiento,
		&p.Direccion,
		&p.Notas,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &p, nil
}

func (r *repository) ActualizarPerfil(ctx context.Context, idUsuario uuid.UUID, req UpdateClienteRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Actualizar tabla usuarios
	queryUser := `
		UPDATE usuarios 
		SET nombre = $1, apellido = $2, telefono = $3
		WHERE id_usuario = $4
	`
	if _, err := tx.ExecContext(ctx, queryUser, req.Nombre, req.Apellido, req.Telefono, idUsuario); err != nil {
		return err
	}

	// Insertar o actualizar tabla clientes
	queryCliente := `
		INSERT INTO clientes (id_usuario, direccion, fecha_nacimiento)
		VALUES ($1, $2, $3)
		ON CONFLICT (id_usuario) DO UPDATE
		SET direccion = EXCLUDED.direccion, fecha_nacimiento = EXCLUDED.fecha_nacimiento
	`
	if _, err := tx.ExecContext(ctx, queryCliente, idUsuario, req.Direccion, req.FechaNacimiento); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *repository) ListarReservasPorCliente(ctx context.Context, idCliente uuid.UUID) ([]ReservaHistorial, error) {
	query := `
		SELECT 
			r.id_reserva, v.placa, v.modelo, el.codigo, 
			r.fecha_reserva, r.hora_inicio::text, r.hora_fin::text, r.estado, r.total_estimado
		FROM reservas r
		INNER JOIN vehiculos v ON r.id_vehiculo = v.id_vehiculo
		INNER JOIN espacios_lavado el ON r.id_espacio = el.id_espacio
		WHERE r.id_cliente = $1
		ORDER BY r.fecha_reserva DESC, r.hora_inicio DESC
	`
	rows, err := r.db.QueryContext(ctx, query, idCliente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []ReservaHistorial
	for rows.Next() {
		var item ReservaHistorial
		if err := rows.Scan(
			&item.IDReserva,
			&item.PlacaVehiculo,
			&item.Modelo,
			&item.CodigoEspacio,
			&item.FechaReserva,
			&item.HoraInicio,
			&item.HoraFin,
			&item.Estado,
			&item.TotalEstimado,
		); err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	return lista, nil
}

func (r *repository) ListarAtencionesPorCliente(ctx context.Context, idCliente uuid.UUID) ([]AtencionHistorial, error) {
	query := `
		SELECT 
			a.id_atencion, a.id_reserva, v.placa, 
			a.estado, a.fecha_inicio_real, a.fecha_fin_real
		FROM atenciones a
		INNER JOIN reservas r ON a.id_reserva = r.id_reserva
		INNER JOIN vehiculos v ON r.id_vehiculo = v.id_vehiculo
		WHERE r.id_cliente = $1
		ORDER BY a.fecha_inicio_real DESC NULLS LAST
	`
	rows, err := r.db.QueryContext(ctx, query, idCliente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []AtencionHistorial
	for rows.Next() {
		var item AtencionHistorial
		if err := rows.Scan(
			&item.IDAtencion,
			&item.IDReserva,
			&item.PlacaVehiculo,
			&item.Estado,
			&item.FechaInicioReal,
			&item.FechaFinReal,
		); err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	return lista, nil
}

func (r *repository) ListarPagosPorCliente(ctx context.Context, idCliente uuid.UUID) ([]PagoHistorial, error) {
	query := `
		SELECT 
			p.id_pago, p.monto, p.metodo, p.estado, p.comprobante_interno, p.fecha_pago
		FROM pagos p
		LEFT JOIN atenciones a ON p.id_atencion = a.id_atencion
		LEFT JOIN reservas r ON a.id_reserva = r.id_reserva
		LEFT JOIN pedidos ped ON p.id_pedido = ped.id_pedido
		WHERE r.id_cliente = $1 OR ped.id_cliente = $1
		ORDER BY p.fecha_pago DESC
	`
	rows, err := r.db.QueryContext(ctx, query, idCliente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []PagoHistorial
	for rows.Next() {
		var item PagoHistorial
		if err := rows.Scan(
			&item.IDPago,
			&item.Monto,
			&item.Metodo,
			&item.Estado,
			&item.ComprobanteInterno,
			&item.FechaPago,
		); err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	return lista, nil
}

func (r *repository) ListarPedidosPorCliente(ctx context.Context, idCliente uuid.UUID) ([]PedidoHistorial, error) {
	query := `
		SELECT 
			id_pedido, estado, total, observaciones, fecha_registro, fecha_entrega
		FROM pedidos
		WHERE id_cliente = $1
		ORDER BY fecha_registro DESC
	`
	rows, err := r.db.QueryContext(ctx, query, idCliente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []PedidoHistorial
	for rows.Next() {
		var item PedidoHistorial
		if err := rows.Scan(
			&item.IDPedido,
			&item.Estado,
			&item.Total,
			&item.Observaciones,
			&item.FechaRegistro,
			&item.FechaEntrega,
		); err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	return lista, nil
}
