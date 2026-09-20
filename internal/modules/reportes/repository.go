package reportes

import (
	"context"
	"database/sql"

	"carwashinmaribackend/internal/modules/atenciones"
	"carwashinmaribackend/internal/utils"
	"github.com/google/uuid"
)

type Repository interface {
	Consultar(context.Context, uuid.UUID, utils.RangoFechas) (*Dashboard, error)
}

type repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return &repository{db: db} }

const pagosSede = `
WITH pagos_sede AS (
 SELECT p.* FROM pagos p WHERE
 EXISTS(SELECT 1 FROM atenciones a JOIN reservas r ON r.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=r.id_cliente
        WHERE a.id_atencion=p.id_atencion AND u.id_sede=$1)
 OR EXISTS(SELECT 1 FROM pedidos pe JOIN usuarios u ON u.id_usuario=pe.id_cliente
        WHERE pe.id_pedido=p.id_pedido AND u.id_sede=$1)
) `

func (r *repository) Consultar(ctx context.Context, sede uuid.UUID, rango utils.RangoFechas) (*Dashboard, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	d := &Dashboard{
		Servicios:     []Servicio{},
		Productividad: []atenciones.RendimientoEquipo{},
		Vehiculos:     []TipoVehiculo{},
		Notas: []string{
			"Los ingresos brutos se reconocen en la fecha del pago y los reembolsos en la fecha de reversión.",
			"El valor de servicios usa el precio reservado y es una métrica operativa; no reemplaza los ingresos cobrados.",
			"La demanda usa la programación actual de cada reserva; el esquema no conserva el historial de fechas reprogramadas.",
		},
	}
	if err = r.cargarIngresos(ctx, tx, sede, rango, &d.Ingresos); err != nil {
		return nil, err
	}
	if d.Servicios, err = r.cargarServicios(ctx, tx, sede, rango); err != nil {
		return nil, err
	}
	if err = r.cargarVentas(ctx, tx, sede, rango, &d.Ventas); err != nil {
		return nil, err
	}
	if err = r.cargarReservas(ctx, tx, sede, rango, &d.Reservas); err != nil {
		return nil, err
	}
	if err = r.cargarDemanda(ctx, tx, sede, rango, &d.Demanda); err != nil {
		return nil, err
	}
	if d.Productividad, err = atenciones.ConsultarRendimientoEquipo(ctx, tx, sede, nil, rango); err != nil {
		return nil, err
	}
	if d.Vehiculos, err = r.cargarVehiculos(ctx, tx, sede, rango); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *repository) cargarIngresos(ctx context.Context, q *sql.Tx, sede uuid.UUID, rango utils.RangoFechas, out *Ingresos) error {
	if err := q.QueryRowContext(ctx, pagosSede+`
 SELECT COALESCE(sum(monto) FILTER(WHERE estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3),0)::text,
        COALESCE(sum(monto) FILTER(WHERE estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3),0)::text,
        (COALESCE(sum(monto) FILTER(WHERE estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3),0)
        -COALESCE(sum(monto) FILTER(WHERE estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3),0))::text
 FROM pagos_sede`, sede, rango.Desde, rango.Hasta).Scan(&out.Brutos, &out.Reembolsados, &out.Netos); err != nil {
		return err
	}

	rows, err := q.QueryContext(ctx, pagosSede+`
 SELECT CASE WHEN id_atencion IS NOT NULL THEN 'Servicios' ELSE 'Productos' END,
        COALESCE(sum(monto) FILTER(WHERE estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3),0)::text,
        COALESCE(sum(monto) FILTER(WHERE estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3),0)::text,
        (COALESCE(sum(monto) FILTER(WHERE estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3),0)-
         COALESCE(sum(monto) FILTER(WHERE estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3),0))::text
 FROM pagos_sede WHERE (estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3)
 OR (estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3)
 GROUP BY 1 ORDER BY 1`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return err
	}
	out.PorTipo = []MovimientoCategoria{}
	for rows.Next() {
		var x MovimientoCategoria
		if err = rows.Scan(&x.Categoria, &x.Bruto, &x.Reembolso, &x.Neto); err != nil {
			rows.Close()
			return err
		}
		out.PorTipo = append(out.PorTipo, x)
	}
	if err = rows.Close(); err != nil {
		return err
	}

	rows, err = q.QueryContext(ctx, pagosSede+`
 SELECT metodo::text,
        COALESCE(sum(monto) FILTER(WHERE estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3),0)::text,
        COALESCE(sum(monto) FILTER(WHERE estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3),0)::text,
        (COALESCE(sum(monto) FILTER(WHERE estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3),0)-
         COALESCE(sum(monto) FILTER(WHERE estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3),0))::text
 FROM pagos_sede WHERE (estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3)
 OR (estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3)
 GROUP BY metodo ORDER BY metodo`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return err
	}
	out.PorMetodo = []MovimientoCategoria{}
	for rows.Next() {
		var x MovimientoCategoria
		if err = rows.Scan(&x.Categoria, &x.Bruto, &x.Reembolso, &x.Neto); err != nil {
			rows.Close()
			return err
		}
		out.PorMetodo = append(out.PorMetodo, x)
	}
	if err = rows.Close(); err != nil {
		return err
	}

	rows, err = q.QueryContext(ctx, pagosSede+`, eventos AS (
 SELECT (fecha_pago AT TIME ZONE 'America/Lima')::date fecha,monto bruto,0::numeric reembolso
 FROM pagos_sede WHERE estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3
 UNION ALL
 SELECT (fecha_reversion AT TIME ZONE 'America/Lima')::date,0::numeric,monto
 FROM pagos_sede WHERE estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3
) SELECT fecha::text,sum(bruto)::text,sum(reembolso)::text,(sum(bruto)-sum(reembolso))::text
 FROM eventos GROUP BY fecha ORDER BY fecha`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return err
	}
	out.Serie = []PuntoFinanciero{}
	for rows.Next() {
		var x PuntoFinanciero
		if err = rows.Scan(&x.Fecha, &x.Bruto, &x.Reembolsado, &x.Neto); err != nil {
			rows.Close()
			return err
		}
		out.Serie = append(out.Serie, x)
	}
	return rows.Close()
}

func (r *repository) cargarServicios(ctx context.Context, q *sql.Tx, sede uuid.UUID, rango utils.RangoFechas) ([]Servicio, error) {
	rows, err := q.QueryContext(ctx, `SELECT s.id_servicio::text,s.nombre,sum(rs.cantidad)::int,
 COALESCE(sum(rs.precio_unitario*rs.cantidad),0)::text
 FROM atenciones a JOIN reservas r ON r.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=r.id_cliente
 JOIN reserva_servicios rs ON rs.id_reserva=r.id_reserva JOIN servicios s ON s.id_servicio=rs.id_servicio
 WHERE u.id_sede=$1 AND a.estado IN ('finalizada','entregada') AND r.estado='completada'
 AND a.fecha_fin_real >= $2 AND a.fecha_fin_real < $3
 GROUP BY s.id_servicio,s.nombre ORDER BY sum(rs.cantidad) DESC,s.nombre`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Servicio{}
	for rows.Next() {
		var x Servicio
		if err = rows.Scan(&x.IDServicio, &x.Nombre, &x.Cantidad, &x.Valor); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *repository) cargarVentas(ctx context.Context, q *sql.Tx, sede uuid.UUID, rango utils.RangoFechas, out *Ventas) error {
	rows, err := q.QueryContext(ctx, `SELECT pe.estado::text,count(*)::int FROM pedidos pe JOIN usuarios u ON u.id_usuario=pe.id_cliente
 WHERE u.id_sede=$1 AND pe.fecha_registro >= $2 AND pe.fecha_registro < $3 GROUP BY pe.estado ORDER BY pe.estado`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return err
	}
	out.PedidosPorEstado = []EstadoConteo{}
	for rows.Next() {
		var x EstadoConteo
		if err = rows.Scan(&x.Estado, &x.Cantidad); err != nil {
			rows.Close()
			return err
		}
		out.PedidosPorEstado = append(out.PedidosPorEstado, x)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	if err = q.QueryRowContext(ctx, `SELECT COALESCE(sum(dp.cantidad),0)::int FROM pedidos pe JOIN usuarios u ON u.id_usuario=pe.id_cliente
 JOIN detalle_pedido dp ON dp.id_pedido=pe.id_pedido WHERE u.id_sede=$1 AND pe.estado IN ('pagado','preparando','entregado')
 AND pe.fecha_registro >= $2 AND pe.fecha_registro < $3`, sede, rango.Desde, rango.Hasta).Scan(&out.UnidadesVendidas); err != nil {
		return err
	}
	return q.QueryRowContext(ctx, pagosSede+` SELECT
 COALESCE(sum(monto) FILTER(WHERE id_pedido IS NOT NULL AND estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3),0)::text,
 COALESCE(sum(monto) FILTER(WHERE id_pedido IS NOT NULL AND estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3),0)::text,
 (COALESCE(sum(monto) FILTER(WHERE id_pedido IS NOT NULL AND estado IN ('pagado','reembolsado') AND fecha_pago >= $2 AND fecha_pago < $3),0)-
 COALESCE(sum(monto) FILTER(WHERE id_pedido IS NOT NULL AND estado='reembolsado' AND fecha_reversion >= $2 AND fecha_reversion < $3),0))::text
 FROM pagos_sede`, sede, rango.Desde, rango.Hasta).Scan(&out.CobradoBruto, &out.Reembolsado, &out.CobradoNeto)
}

func (r *repository) cargarReservas(ctx context.Context, q *sql.Tx, sede uuid.UUID, rango utils.RangoFechas, out *Reservas) error {
	rows, err := q.QueryContext(ctx, `SELECT r.estado::text,count(*)::int FROM reservas r JOIN usuarios u ON u.id_usuario=r.id_cliente
 WHERE u.id_sede=$1 AND r.fecha_reserva >= ($2 AT TIME ZONE 'America/Lima')::date
 AND r.fecha_reserva < ($3 AT TIME ZONE 'America/Lima')::date GROUP BY r.estado ORDER BY r.estado`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return err
	}
	out.PorEstado = []EstadoConteo{}
	for rows.Next() {
		var x EstadoConteo
		if err = rows.Scan(&x.Estado, &x.Cantidad); err != nil {
			rows.Close()
			return err
		}
		out.PorEstado = append(out.PorEstado, x)
		out.Total += x.Cantidad
		if x.Estado == "cancelada" {
			out.Canceladas = x.Cantidad
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}
	if out.Total > 0 {
		out.TasaCancelacion = float64(out.Canceladas) * 100 / float64(out.Total)
	}
	return nil
}

func (r *repository) cargarDemanda(ctx context.Context, q *sql.Tx, sede uuid.UUID, rango utils.RangoFechas, out *Demanda) error {
	rows, err := q.QueryContext(ctx, `SELECT r.fecha_reserva::text,count(*)::int,count(*) FILTER(WHERE r.estado='cancelada')::int
 FROM reservas r JOIN usuarios u ON u.id_usuario=r.id_cliente WHERE u.id_sede=$1
 AND r.fecha_reserva >= ($2 AT TIME ZONE 'America/Lima')::date AND r.fecha_reserva < ($3 AT TIME ZONE 'America/Lima')::date
 GROUP BY r.fecha_reserva ORDER BY r.fecha_reserva`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return err
	}
	out.PorDia = []DemandaDia{}
	for rows.Next() {
		var x DemandaDia
		if err = rows.Scan(&x.Fecha, &x.Total, &x.Canceladas); err != nil {
			rows.Close()
			return err
		}
		out.PorDia = append(out.PorDia, x)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	rows, err = q.QueryContext(ctx, `SELECT CASE WHEN r.hora_inicio<'09:00' THEN 'Antes de 09:00' WHEN r.hora_inicio<'12:00' THEN '09:00–11:59'
 WHEN r.hora_inicio<'15:00' THEN '12:00–14:59' WHEN r.hora_inicio<'18:00' THEN '15:00–17:59' ELSE '18:00 en adelante' END franja,
 count(*)::int,count(*) FILTER(WHERE r.estado='cancelada')::int FROM reservas r JOIN usuarios u ON u.id_usuario=r.id_cliente
 WHERE u.id_sede=$1 AND r.fecha_reserva >= ($2 AT TIME ZONE 'America/Lima')::date AND r.fecha_reserva < ($3 AT TIME ZONE 'America/Lima')::date
 GROUP BY franja ORDER BY min(r.hora_inicio)`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return err
	}
	out.PorHorario = []DemandaCategoria{}
	for rows.Next() {
		var x DemandaCategoria
		if err = rows.Scan(&x.Categoria, &x.Total, &x.Canceladas); err != nil {
			rows.Close()
			return err
		}
		out.PorHorario = append(out.PorHorario, x)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	rows, err = q.QueryContext(ctx, `SELECT s.nombre,sum(rs.cantidad)::int,COALESCE(sum(rs.cantidad) FILTER(WHERE r.estado='cancelada'),0)::int
 FROM reservas r JOIN usuarios u ON u.id_usuario=r.id_cliente JOIN reserva_servicios rs ON rs.id_reserva=r.id_reserva
 JOIN servicios s ON s.id_servicio=rs.id_servicio WHERE u.id_sede=$1
 AND r.fecha_reserva >= ($2 AT TIME ZONE 'America/Lima')::date AND r.fecha_reserva < ($3 AT TIME ZONE 'America/Lima')::date
 GROUP BY s.id_servicio,s.nombre ORDER BY sum(rs.cantidad) DESC,s.nombre`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return err
	}
	out.PorServicio = []DemandaCategoria{}
	for rows.Next() {
		var x DemandaCategoria
		if err = rows.Scan(&x.Categoria, &x.Total, &x.Canceladas); err != nil {
			rows.Close()
			return err
		}
		out.PorServicio = append(out.PorServicio, x)
	}
	return rows.Close()
}

func (r *repository) cargarVehiculos(ctx context.Context, q *sql.Tx, sede uuid.UUID, rango utils.RangoFechas) ([]TipoVehiculo, error) {
	rows, err := q.QueryContext(ctx, `SELECT COALESCE(NULLIF(btrim(v.tipo_vehiculo),''),'Sin clasificar'),count(*)::int,count(DISTINCT v.id_vehiculo)::int
 FROM atenciones a JOIN reservas r ON r.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=r.id_cliente
 JOIN vehiculos v ON v.id_vehiculo=r.id_vehiculo WHERE u.id_sede=$1 AND a.estado IN ('finalizada','entregada') AND r.estado='completada'
 AND a.fecha_fin_real >= $2 AND a.fecha_fin_real < $3 GROUP BY 1 ORDER BY count(*) DESC,1`, sede, rango.Desde, rango.Hasta)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TipoVehiculo{}
	for rows.Next() {
		var x TipoVehiculo
		if err = rows.Scan(&x.Tipo, &x.Atenciones, &x.VehiculosDistintos); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
