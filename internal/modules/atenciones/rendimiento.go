package atenciones

import (
	"context"
	"database/sql"

	"carwashinmaribackend/internal/utils"
	"github.com/google/uuid"
)

type metricasQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// ConsultarRendimientoEquipo contiene la única definición de productividad
// usada por RF-13 y RF-14. Una persona cuenta una vez por atención aunque una
// asignación se haya duplicado, y los servicios se reparten como participación
// del equipo, no como ingreso individual.
func ConsultarRendimientoEquipo(ctx context.Context, q metricasQueryer, sede uuid.UUID, trabajador *uuid.UUID, rango utils.RangoFechas) ([]RendimientoEquipo, error) {
	rows, err := q.QueryContext(ctx, `
WITH asignaciones_unicas AS (
    SELECT DISTINCT id_trabajador,id_atencion
    FROM asignaciones_trabajadores
), servicios_atencion AS (
    SELECT a.id_atencion,COALESCE(sum(rs.cantidad),0)::int AS cantidad
    FROM atenciones a
    JOIN reserva_servicios rs ON rs.id_reserva=a.id_reserva
    GROUP BY a.id_atencion
)
SELECT t.id_usuario,u.nombre||' '||u.apellido,
       count(*)::int,COALESCE(sum(sa.cantidad),0)::int,
       avg(CASE WHEN a.fecha_fin_real>=a.fecha_inicio_real
           THEN extract(epoch FROM (a.fecha_fin_real-a.fecha_inicio_real))/60 END)::float8
FROM asignaciones_unicas au
JOIN trabajadores t ON t.id_usuario=au.id_trabajador
JOIN usuarios u ON u.id_usuario=t.id_usuario
JOIN atenciones a ON a.id_atencion=au.id_atencion
JOIN reservas r ON r.id_reserva=a.id_reserva
LEFT JOIN servicios_atencion sa ON sa.id_atencion=a.id_atencion
WHERE u.id_sede=$1
  AND a.estado IN ('finalizada','entregada') AND r.estado='completada'
  AND ($2::uuid IS NULL OR t.id_usuario=$2)
  AND ($3::timestamptz IS NULL OR a.fecha_fin_real >= $3)
  AND ($4::timestamptz IS NULL OR a.fecha_fin_real < $4)
GROUP BY t.id_usuario,u.nombre,u.apellido
ORDER BY count(*) DESC,COALESCE(sum(sa.cantidad),0) DESC,u.nombre,u.apellido`, sede, trabajador, rango.Desde, rango.Hasta)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultado := []RendimientoEquipo{}
	for rows.Next() {
		var fila RendimientoEquipo
		if err = rows.Scan(&fila.IDTrabajador, &fila.Trabajador, &fila.Atenciones, &fila.Servicios, &fila.DuracionPromedio); err != nil {
			return nil, err
		}
		resultado = append(resultado, fila)
	}
	return resultado, rows.Err()
}
