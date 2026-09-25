package pagos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"carwashinmaribackend/internal/utils"
	"github.com/google/uuid"
)

type repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return &repository{db: db} }

const selectPago = `
 SELECT p.id_pago,p.id_atencion,p.id_pedido,
 CASE WHEN p.id_atencion IS NOT NULL THEN 'atencion' ELSE 'pedido' END,
 p.monto::text,p.moneda,p.metodo::text,p.estado::text,p.comprobante_interno,
 p.referencia_externa,p.fecha_pago,p.fecha_reversion,p.motivo_reversion,
 COALESCE(p.detalle_snapshot->>'cliente',''),p.detalle_snapshot
 FROM pagos p`

type scanner interface{ Scan(...any) error }

func scanPago(row scanner, p *Pago) error {
	return row.Scan(&p.IDPago, &p.IDAtencion, &p.IDPedido, &p.Tipo, &p.Monto, &p.Moneda,
		&p.Metodo, &p.Estado, &p.ComprobanteInterno, &p.ReferenciaExterna, &p.FechaPago,
		&p.FechaReversion, &p.MotivoReversion, &p.Cliente, &p.Detalle)
}

func (r *repository) Listar(ctx context.Context, sede uuid.UUID, estado string) ([]Pago, error) {
	rows, err := r.db.QueryContext(ctx, selectPago+`
 WHERE ($2='' OR p.estado::text=$2) AND (
  EXISTS(SELECT 1 FROM atenciones a JOIN reservas re ON re.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=re.id_cliente WHERE a.id_atencion=p.id_atencion AND u.id_sede=$1)
  OR EXISTS(SELECT 1 FROM pedidos pe JOIN usuarios u ON u.id_usuario=pe.id_cliente WHERE pe.id_pedido=p.id_pedido AND u.id_sede=$1))
 ORDER BY p.fecha_pago DESC,p.id_pago`, sede, estado)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []Pago{}
	for rows.Next() {
		var p Pago
		if err = scanPago(rows, &p); err != nil {
			return nil, err
		}
		lista = append(lista, p)
	}
	return lista, rows.Err()
}

func (r *repository) ListarPorCliente(ctx context.Context, cliente uuid.UUID) ([]Pago, error) {
	rows, err := r.db.QueryContext(ctx, selectPago+` WHERE
 EXISTS(SELECT 1 FROM atenciones a JOIN reservas re ON re.id_reserva=a.id_reserva WHERE a.id_atencion=p.id_atencion AND re.id_cliente=$1)
 OR EXISTS(SELECT 1 FROM pedidos pe WHERE pe.id_pedido=p.id_pedido AND pe.id_cliente=$1)
 ORDER BY p.fecha_pago DESC,p.id_pago`, cliente)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []Pago{}
	for rows.Next() {
		var p Pago
		if err = scanPago(rows, &p); err != nil {
			return nil, err
		}
		lista = append(lista, p)
	}
	return lista, rows.Err()
}

func (r *repository) ListarPendientes(ctx context.Context, sede uuid.UUID, tipo string) ([]OperacionPendiente, error) {
	rows, err := r.db.QueryContext(ctx, `
 SELECT 'atencion',a.id_atencion,u.nombre||' '||u.apellido,v.placa||' · servicios finalizados',
 COALESCE(sum(rs.precio_unitario*rs.cantidad),0)::text,a.fecha_fin_real,
 jsonb_build_object('cliente',u.nombre||' '||u.apellido,'placa',v.placa,'id_reserva',re.id_reserva,'fecha_reserva',re.fecha_reserva,'servicios',
   COALESCE(jsonb_agg(jsonb_build_object('nombre',s.nombre,'cantidad',rs.cantidad,'precio_unitario',rs.precio_unitario) ORDER BY s.nombre),'[]'::jsonb))
 FROM atenciones a JOIN reservas re ON re.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=re.id_cliente
 JOIN vehiculos v ON v.id_vehiculo=re.id_vehiculo JOIN reserva_servicios rs ON rs.id_reserva=re.id_reserva JOIN servicios s ON s.id_servicio=rs.id_servicio
 WHERE u.id_sede=$1 AND a.estado IN ('finalizada','entregada') AND re.estado='completada'
 AND NOT EXISTS(SELECT 1 FROM pagos p WHERE p.id_atencion=a.id_atencion AND p.estado='pagado')
 AND ($2='' OR $2='atencion') GROUP BY a.id_atencion,re.id_reserva,re.fecha_reserva,u.nombre,u.apellido,v.placa,a.fecha_fin_real
 UNION ALL
 SELECT 'pedido',pe.id_pedido,u.nombre||' '||u.apellido,'Pedido de productos',
 COALESCE(sum(dp.precio_unitario*dp.cantidad),0)::text,pe.fecha_registro,
 jsonb_build_object('cliente',u.nombre||' '||u.apellido,'productos',
   COALESCE(jsonb_agg(jsonb_build_object('nombre',pr.nombre,'cantidad',dp.cantidad,'precio_unitario',dp.precio_unitario) ORDER BY pr.nombre),'[]'::jsonb))
 FROM pedidos pe JOIN usuarios u ON u.id_usuario=pe.id_cliente JOIN detalle_pedido dp ON dp.id_pedido=pe.id_pedido JOIN productos pr ON pr.id_producto=dp.id_producto
 WHERE u.id_sede=$1 AND pe.estado='registrado'
 AND NOT EXISTS(SELECT 1 FROM pagos p WHERE p.id_pedido=pe.id_pedido AND p.estado='pagado')
 AND ($2='' OR $2='pedido') GROUP BY pe.id_pedido,u.nombre,u.apellido,pe.fecha_registro
 ORDER BY 6 DESC`, sede, tipo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []OperacionPendiente{}
	for rows.Next() {
		var o OperacionPendiente
		o.Moneda = "PEN"
		if err = rows.Scan(&o.Tipo, &o.IDOperacion, &o.Cliente, &o.Descripcion, &o.Monto, &o.Fecha, &o.Detalle); err != nil {
			return nil, err
		}
		lista = append(lista, o)
	}
	return lista, rows.Err()
}

func comprobante(id uuid.UUID, ahora time.Time) string {
	return fmt.Sprintf("CI-%s-%s", ahora.In(utils.ZonaNegocio).Format("20060102"), strings.ToUpper(strings.ReplaceAll(id.String(), "-", "")[:12]))
}

func (r *repository) Registrar(ctx context.Context, sede, admin uuid.UUID, in RegistrarPagoInput, requestHash string) (*Pago, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var existente Pago
	err = scanPago(tx.QueryRowContext(ctx, selectPago+` WHERE p.registrado_por=$1 AND p.idempotency_key=$2 FOR UPDATE`, admin, in.IdempotencyKey), &existente)
	if err == nil {
		var previo string
		if err = tx.QueryRowContext(ctx, `SELECT request_hash FROM pagos WHERE id_pago=$1`, existente.IDPago).Scan(&previo); err != nil {
			return nil, err
		}
		if previo != requestHash {
			return nil, utils.Conflict("la clave de idempotencia ya fue usada con otros datos")
		}
		return &existente, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	var monto string
	var detalle []byte
	var idAtencion, idPedido *uuid.UUID
	if in.Tipo == TipoAtencion {
		var id uuid.UUID
		var estadoAtencion, estadoReserva string
		err = tx.QueryRowContext(ctx, `SELECT a.id_atencion,a.estado::text,re.estado::text
 FROM atenciones a JOIN reservas re ON re.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=re.id_cliente
 WHERE a.id_atencion=$1 AND u.id_sede=$2 FOR UPDATE OF a,re`, in.IDOperacion, sede).Scan(&id, &estadoAtencion, &estadoReserva)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, utils.NotFound("atención no encontrada")
		}
		if err != nil {
			return nil, err
		}
		if estadoAtencion != "finalizada" && estadoAtencion != "entregada" || estadoReserva != "completada" {
			return nil, utils.Conflict("la atención debe estar finalizada antes de cobrar")
		}
		err = tx.QueryRowContext(ctx, `SELECT COALESCE(sum(rs.precio_unitario*rs.cantidad),0)::text,
 jsonb_build_object('cliente',u.nombre||' '||u.apellido,'correo',u.correo,'placa',v.placa,'id_reserva',re.id_reserva,'fecha_reserva',re.fecha_reserva,
 'servicios',jsonb_agg(jsonb_build_object('nombre',s.nombre,'cantidad',rs.cantidad,'precio_unitario',rs.precio_unitario) ORDER BY s.nombre))
 FROM atenciones a JOIN reservas re ON re.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=re.id_cliente
 JOIN vehiculos v ON v.id_vehiculo=re.id_vehiculo JOIN reserva_servicios rs ON rs.id_reserva=re.id_reserva JOIN servicios s ON s.id_servicio=rs.id_servicio
 WHERE a.id_atencion=$1 GROUP BY re.id_reserva,re.fecha_reserva,u.nombre,u.apellido,u.correo,v.placa`, in.IDOperacion).Scan(&monto, &detalle)
		if err != nil {
			return nil, err
		}
		idAtencion = &id
	} else {
		var id uuid.UUID
		var estado string
		err = tx.QueryRowContext(ctx, `SELECT pe.id_pedido,pe.estado::text FROM pedidos pe JOIN usuarios u ON u.id_usuario=pe.id_cliente
 WHERE pe.id_pedido=$1 AND u.id_sede=$2 FOR UPDATE OF pe`, in.IDOperacion, sede).Scan(&id, &estado)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, utils.NotFound("pedido no encontrado")
		}
		if err != nil {
			return nil, err
		}
		if estado != "registrado" {
			return nil, utils.Conflict("solo se pueden cobrar pedidos registrados")
		}
		err = tx.QueryRowContext(ctx, `SELECT COALESCE(sum(dp.precio_unitario*dp.cantidad),0)::text,
 jsonb_build_object('cliente',u.nombre||' '||u.apellido,'correo',u.correo,'productos',
 jsonb_agg(jsonb_build_object('nombre',pr.nombre,'cantidad',dp.cantidad,'precio_unitario',dp.precio_unitario) ORDER BY pr.nombre))
 FROM pedidos pe JOIN usuarios u ON u.id_usuario=pe.id_cliente JOIN detalle_pedido dp ON dp.id_pedido=pe.id_pedido JOIN productos pr ON pr.id_producto=dp.id_producto
 WHERE pe.id_pedido=$1 GROUP BY u.nombre,u.apellido,u.correo`, in.IDOperacion).Scan(&monto, &detalle)
		if err != nil {
			return nil, err
		}
		idPedido = &id
	}
	if monto == "0.00" || monto == "0" {
		return nil, utils.Conflict("la operación no tiene un importe cobrable")
	}
	id := uuid.New()
	ahora := time.Now()
	numero := comprobante(id, ahora)
	_, err = tx.ExecContext(ctx, `INSERT INTO pagos(id_pago,id_atencion,id_pedido,monto,metodo,estado,comprobante_interno,fecha_pago,moneda,
 referencia_externa,registrado_por,idempotency_key,request_hash,detalle_snapshot)
 VALUES($1,$2,$3,$4::numeric,$5::metodo_pago,'pagado',$6,$7,'PEN',$8,$9,$10,$11,$12)`, id, idAtencion, idPedido, monto, in.Metodo, numero, ahora, in.ReferenciaExterna, admin, in.IdempotencyKey, requestHash, detalle)
	if err != nil {
		if !utils.IsUniqueViolation(err) {
			return nil, err
		}
		_ = tx.Rollback()
		var repetido Pago
		if scanPago(r.db.QueryRowContext(ctx, selectPago+` WHERE p.registrado_por=$1 AND p.idempotency_key=$2`, admin, in.IdempotencyKey), &repetido) == nil {
			var previo string
			if r.db.QueryRowContext(ctx, `SELECT request_hash FROM pagos WHERE id_pago=$1`, repetido.IDPago).Scan(&previo) == nil && previo == requestHash {
				return &repetido, nil
			}
			return nil, utils.Conflict("la clave de idempotencia ya fue usada con otros datos")
		}
		return nil, utils.Conflict("la operación ya tiene un pago vigente")
	}
	if idPedido != nil {
		if _, err = tx.ExecContext(ctx, `UPDATE pedidos SET estado='pagado' WHERE id_pedido=$1 AND estado='registrado'`, *idPedido); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.ObtenerComprobante(ctx, utils.CustomClaims{UserID: admin, SedeID: sede, Rol: "administrador"}, id)
}

func (r *repository) ObtenerComprobante(ctx context.Context, actor utils.CustomClaims, id uuid.UUID) (*Pago, error) {
	var p Pago
	err := scanPago(r.db.QueryRowContext(ctx, selectPago+` WHERE p.id_pago=$1 AND (
 ($2='administrador' AND (EXISTS(SELECT 1 FROM atenciones a JOIN reservas re ON re.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=re.id_cliente WHERE a.id_atencion=p.id_atencion AND u.id_sede=$4)
 OR EXISTS(SELECT 1 FROM pedidos pe JOIN usuarios u ON u.id_usuario=pe.id_cliente WHERE pe.id_pedido=p.id_pedido AND u.id_sede=$4)))
 OR ($2='cliente' AND (EXISTS(SELECT 1 FROM atenciones a JOIN reservas re ON re.id_reserva=a.id_reserva WHERE a.id_atencion=p.id_atencion AND re.id_cliente=$3)
 OR EXISTS(SELECT 1 FROM pedidos pe WHERE pe.id_pedido=p.id_pedido AND pe.id_cliente=$3))))`, id, actor.Rol, actor.UserID, actor.SedeID), &p)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, utils.NotFound("comprobante no encontrado")
	}
	return &p, err
}

func (r *repository) Revertir(ctx context.Context, sede, admin, id uuid.UUID, in RevertirPagoInput) (*Pago, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var idPedido *uuid.UUID
	var estado string
	err = tx.QueryRowContext(ctx, `SELECT p.id_pedido,p.estado::text FROM pagos p WHERE p.id_pago=$1 AND (
 EXISTS(SELECT 1 FROM atenciones a JOIN reservas re ON re.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=re.id_cliente WHERE a.id_atencion=p.id_atencion AND u.id_sede=$2)
 OR EXISTS(SELECT 1 FROM pedidos pe JOIN usuarios u ON u.id_usuario=pe.id_cliente WHERE pe.id_pedido=p.id_pedido AND u.id_sede=$2)) FOR UPDATE OF p`, id, sede).Scan(&idPedido, &estado)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, utils.NotFound("pago no encontrado")
	}
	if err != nil {
		return nil, err
	}
	if estado != Pagado {
		return nil, utils.Conflict("el pago ya no está vigente")
	}
	if idPedido != nil {
		var estadoPedido string
		if err = tx.QueryRowContext(ctx, `SELECT estado::text FROM pedidos WHERE id_pedido=$1 FOR UPDATE`, *idPedido).Scan(&estadoPedido); err != nil {
			return nil, err
		}
		switch estadoPedido {
		case "pagado":
			_, err = tx.ExecContext(ctx, `UPDATE pedidos SET estado='registrado' WHERE id_pedido=$1`, *idPedido)
		case "preparando":
			_, err = tx.ExecContext(ctx, `UPDATE productos p SET stock=p.stock+d.cantidad FROM detalle_pedido d WHERE d.id_pedido=$1 AND p.id_producto=d.id_producto`, *idPedido)
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE pedidos SET estado='cancelado' WHERE id_pedido=$1`, *idPedido)
			}
		case "entregado":
			return nil, utils.Conflict("un pedido entregado debe gestionarse mediante un reclamo")
		default:
			return nil, utils.Conflict("el estado actual del pedido no admite reversión")
		}
		if err != nil {
			return nil, err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE pagos SET estado='reembolsado',revertido_por=$2,motivo_reversion=$3,fecha_reversion=now() WHERE id_pago=$1`, id, admin, in.Motivo)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.ObtenerComprobante(ctx, utils.CustomClaims{UserID: admin, SedeID: sede, Rol: "administrador"}, id)
}
