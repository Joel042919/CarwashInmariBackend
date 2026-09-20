package pedidos

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

type Repository interface {
	CrearPedido(ctx context.Context, idCliente uuid.UUID, items []ItemPedido, observaciones *string) (*Pedido, error)
	ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]Pedido, error)
	ListarTodos(ctx context.Context, estado string) ([]Pedido, error)
	ObtenerPorID(ctx context.Context, id uuid.UUID) (*Pedido, error)
	CambiarEstado(ctx context.Context, id uuid.UUID, nuevo string, idClienteRestringido *uuid.UUID) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func redondear(v float64) float64 { return math.Round(v*100) / 100 }

// CrearPedido registra el pedido, descuenta stock y calcula el total en una sola
// transacción. El descuento es atómico (UPDATE ... WHERE stock >= cantidad), por
// lo que dos pedidos simultáneos no pueden dejar el stock en negativo.
func (r *repository) CrearPedido(ctx context.Context, idCliente uuid.UUID, items []ItemPedido, observaciones *string) (*Pedido, error) {
	// Orden estable para evitar deadlocks entre pedidos concurrentes.
	sort.Slice(items, func(i, j int) bool { return items[i].IDProducto.String() < items[j].IDProducto.String() })

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	pedido := &Pedido{
		IDPedido:      uuid.New(),
		IDCliente:     idCliente,
		Estado:        EstadoRegistrado,
		Observaciones: observaciones,
		FechaRegistro: time.Now(),
		Detalle:       []DetallePedido{},
	}

	// La FK de detalle_pedido es inmediata: el pedido debe existir antes que su detalle.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pedidos (id_pedido, id_cliente, estado, total, observaciones, fecha_registro)
		VALUES ($1, $2, $3::estado_pedido, 0, $4, $5)`,
		pedido.IDPedido, idCliente, pedido.Estado, observaciones, pedido.FechaRegistro); err != nil {
		return nil, err
	}

	var total float64
	for _, it := range items {
		var nombre string
		var precio float64
		err := tx.QueryRowContext(ctx, `
			UPDATE productos SET stock = stock - $1
			WHERE id_producto = $2 AND activo AND stock >= $1
			RETURNING nombre, precio_venta`, it.Cantidad, it.IDProducto).Scan(&nombre, &precio)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, utils.BadRequest("uno de los productos no existe, no está disponible o no tiene stock suficiente")
			}
			return nil, err
		}

		det := DetallePedido{
			IDDetalle:      uuid.New(),
			IDProducto:     it.IDProducto,
			NombreProducto: nombre,
			Cantidad:       it.Cantidad,
			PrecioUnitario: precio,
			Subtotal:       redondear(precio * float64(it.Cantidad)),
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO detalle_pedido (id_detalle, id_pedido, id_producto, cantidad, precio_unitario, subtotal)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			det.IDDetalle, pedido.IDPedido, det.IDProducto, det.Cantidad, det.PrecioUnitario, det.Subtotal); err != nil {
			return nil, err
		}
		total += det.Subtotal
		pedido.Detalle = append(pedido.Detalle, det)
	}

	pedido.Total = redondear(total)
	if _, err := tx.ExecContext(ctx, `UPDATE pedidos SET total = $1 WHERE id_pedido = $2`, pedido.Total, pedido.IDPedido); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return pedido, nil
}

const selectPedido = `
	SELECT p.id_pedido, p.id_cliente, u.nombre || ' ' || u.apellido, u.correo,
		p.estado::text, p.total, p.observaciones, p.fecha_registro, p.fecha_entrega
	FROM pedidos p
	INNER JOIN usuarios u ON u.id_usuario = p.id_cliente
`

type scanner interface {
	Scan(dest ...any) error
}

func scanPedido(sc scanner, p *Pedido) error {
	return sc.Scan(
		&p.IDPedido, &p.IDCliente, &p.NombreCliente, &p.CorreoCliente,
		&p.Estado, &p.Total, &p.Observaciones, &p.FechaRegistro, &p.FechaEntrega,
	)
}

func (r *repository) cargarDetalle(ctx context.Context, p *Pedido) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id_detalle, d.id_producto, pr.nombre, d.cantidad, d.precio_unitario, d.subtotal
		FROM detalle_pedido d
		INNER JOIN productos pr ON pr.id_producto = d.id_producto
		WHERE d.id_pedido = $1
		ORDER BY pr.nombre`, p.IDPedido)
	if err != nil {
		return err
	}
	defer rows.Close()

	p.Detalle = []DetallePedido{}
	for rows.Next() {
		var d DetallePedido
		if err := rows.Scan(&d.IDDetalle, &d.IDProducto, &d.NombreProducto, &d.Cantidad, &d.PrecioUnitario, &d.Subtotal); err != nil {
			return err
		}
		p.Detalle = append(p.Detalle, d)
	}
	return rows.Err()
}

func (r *repository) listar(ctx context.Context, query string, args ...any) ([]Pedido, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	lista := []Pedido{}
	for rows.Next() {
		var p Pedido
		if err := scanPedido(rows, &p); err != nil {
			rows.Close()
			return nil, err
		}
		lista = append(lista, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range lista {
		if err := r.cargarDetalle(ctx, &lista[i]); err != nil {
			return nil, err
		}
	}
	return lista, nil
}

func (r *repository) ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]Pedido, error) {
	return r.listar(ctx, selectPedido+` WHERE p.id_cliente = $1 ORDER BY p.fecha_registro DESC`, idCliente)
}

func (r *repository) ListarTodos(ctx context.Context, estado string) ([]Pedido, error) {
	return r.listar(ctx, selectPedido+`
		WHERE ($1 = '' OR p.estado::text = $1)
		ORDER BY p.fecha_registro DESC`, estado)
}

func (r *repository) ObtenerPorID(ctx context.Context, id uuid.UUID) (*Pedido, error) {
	var p Pedido
	if err := scanPedido(r.db.QueryRowContext(ctx, selectPedido+` WHERE p.id_pedido = $1`, id), &p); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := r.cargarDetalle(ctx, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

var transiciones = map[string][]string{
	// El estado pagado solo puede establecerlo el módulo de pagos dentro de la
	// misma transacción que registra el comprobante.
	EstadoRegistrado: {EstadoCancelado},
	EstadoPagado:     {EstadoPreparando},
	EstadoPreparando: {EstadoEntregado},
}

func transicionValida(actual, nuevo string) bool {
	for _, e := range transiciones[actual] {
		if e == nuevo {
			return true
		}
	}
	return false
}

// CambiarEstado valida la transición bajo bloqueo de fila. Al cancelar devuelve
// el stock. Si idClienteRestringido no es nil, el pedido debe pertenecerle y solo
// puede cancelarse mientras siga "registrado".
func (r *repository) CambiarEstado(ctx context.Context, id uuid.UUID, nuevo string, idClienteRestringido *uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var actual string
	var idCliente uuid.UUID
	err = tx.QueryRowContext(ctx,
		`SELECT estado::text, id_cliente FROM pedidos WHERE id_pedido = $1 FOR UPDATE`, id).Scan(&actual, &idCliente)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.NotFound("pedido no encontrado")
		}
		return err
	}

	if idClienteRestringido != nil {
		if idCliente != *idClienteRestringido {
			return utils.NotFound("pedido no encontrado")
		}
		if actual != EstadoRegistrado {
			return utils.Conflict("solo puedes cancelar pedidos que aún no fueron pagados")
		}
	}

	if !transicionValida(actual, nuevo) {
		return utils.Conflict("no se puede pasar el pedido de '" + actual + "' a '" + nuevo + "'")
	}

	if nuevo == EstadoCancelado {
		if _, err := tx.ExecContext(ctx, `
			UPDATE productos p SET stock = p.stock + d.cantidad
			FROM detalle_pedido d
			WHERE d.id_pedido = $1 AND p.id_producto = d.id_producto`, id); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE pedidos
		SET estado = $1::text::estado_pedido,
			fecha_entrega = CASE WHEN $1::text = 'entregado' THEN now() ELSE fecha_entrega END
		WHERE id_pedido = $2`, nuevo, id); err != nil {
		return err
	}
	return tx.Commit()
}
