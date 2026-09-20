package atenciones

import (
	"carwashinmaribackend/internal/modules/documentos"
	"carwashinmaribackend/internal/utils"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
)

type Repository interface {
	Listar(context.Context, utils.CustomClaims, Filtro) ([]Atencion, error)
	CambiarEstado(context.Context, utils.CustomClaims, uuid.UUID, string) error
	Historial(context.Context, utils.CustomClaims, uuid.UUID) ([]Evento, error)
	Rendimiento(context.Context, uuid.UUID, uuid.UUID, utils.RangoFechas) (*Rendimiento, error)
}
type repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return &repository{db: db} }

const acceso = ` u.id_sede=$1 AND (
 ($2='administrador') OR ($2='cliente' AND r.id_cliente=$3) OR
 ($2='trabajador' AND EXISTS(SELECT 1 FROM asignaciones_trabajadores x WHERE x.id_atencion=a.id_atencion AND x.id_trabajador=$3))) `
const relaciones = ` FROM atenciones a JOIN reservas r ON r.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=r.id_cliente `

func (repo *repository) Listar(ctx context.Context, c utils.CustomClaims, f Filtro) ([]Atencion, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT a.id_atencion,r.id_reserva,a.estado::text,r.estado::text,
 r.fecha_reserva::text,to_char(r.hora_inicio,'HH24:MI'),to_char(r.hora_fin,'HH24:MI'),v.placa,
 u.nombre || ' ' || u.apellido,a.fecha_inicio_real,a.fecha_fin_real,
 COALESCE((SELECT json_agg(json_build_object('nombre',s.nombre,'cantidad',rs.cantidad) ORDER BY s.nombre)
 FROM reserva_servicios rs JOIN servicios s ON s.id_servicio=rs.id_servicio WHERE rs.id_reserva=r.id_reserva),'[]'::json)
 `+relaciones+` JOIN vehiculos v ON v.id_vehiculo=r.id_vehiculo WHERE `+acceso+`
 AND ($4::uuid IS NULL OR EXISTS(SELECT 1 FROM asignaciones_trabajadores x WHERE x.id_atencion=a.id_atencion AND x.id_trabajador=$4))
 AND ($5='' OR a.estado::text=$5)
 AND ($6::timestamptz IS NULL OR r.fecha_reserva>=($6::timestamptz AT TIME ZONE 'America/Lima')::date)
 AND ($7::timestamptz IS NULL OR r.fecha_reserva<($7::timestamptz AT TIME ZONE 'America/Lima')::date)
 ORDER BY r.fecha_reserva DESC,r.hora_inicio,a.id_atencion`, c.SedeID, c.Rol, c.UserID, f.Trabajador, f.Estado, f.Rango.Desde, f.Rango.Hasta)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []Atencion{}
	for rows.Next() {
		var a Atencion
		var servicios []byte
		if err = rows.Scan(&a.ID, &a.IDReserva, &a.Estado, &a.EstadoReserva, &a.Fecha, &a.HoraInicio, &a.HoraFin, &a.Placa, &a.Cliente, &a.InicioReal, &a.FinReal, &servicios); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(servicios, &a.Servicios); err != nil {
			return nil, err
		}
		lista = append(lista, a)
	}
	return lista, rows.Err()
}

func (repo *repository) Historial(ctx context.Context, c utils.CustomClaims, id uuid.UUID) ([]Evento, error) {
	rows, err := repo.db.QueryContext(ctx, `SELECT h.id_historial,h.estado::text,h.fecha_cambio,autor.nombre || ' ' || autor.apellido,h.comentario
 `+relaciones+` JOIN historial_estados_atencion h ON h.id_atencion=a.id_atencion JOIN usuarios autor ON autor.id_usuario=h.registrado_por
 WHERE `+acceso+` AND a.id_atencion=$4 ORDER BY h.fecha_cambio,h.id_historial`, c.SedeID, c.Rol, c.UserID, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []Evento{}
	for rows.Next() {
		var e Evento
		if err = rows.Scan(&e.ID, &e.Estado, &e.Fecha, &e.Responsable, &e.Comentario); err != nil {
			return nil, err
		}
		lista = append(lista, e)
	}
	return lista, rows.Err()
}

func (repo *repository) CambiarEstado(ctx context.Context, c utils.CustomClaims, id uuid.UUID, nuevo string) error {
	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Todas las modificaciones de programación bloquean primero la reserva.
	var reserva uuid.UUID
	err = tx.QueryRowContext(ctx, `SELECT r.id_reserva `+relaciones+` WHERE `+acceso+` AND a.id_atencion=$4 FOR UPDATE OF r`, c.SedeID, c.Rol, c.UserID, id).Scan(&reserva)
	if errors.Is(err, sql.ErrNoRows) {
		return utils.NotFound("atención no encontrada")
	}
	if err != nil {
		return err
	}
	var actual, estadoReserva string
	err = tx.QueryRowContext(ctx, `SELECT a.estado::text,r.estado::text `+relaciones+` WHERE `+acceso+` AND a.id_atencion=$4 FOR UPDATE OF a`, c.SedeID, c.Rol, c.UserID, id).Scan(&actual, &estadoReserva)
	if errors.Is(err, sql.ErrNoRows) {
		return utils.NotFound("atención no encontrada")
	}
	if err != nil {
		return err
	}
	if err = validarTransicion(actual, nuevo, estadoReserva); err != nil {
		return err
	}
	if nuevo == EnProceso {
		if err = documentos.ExigirDocumentosValidados(ctx, documentos.NewTransactionalChecker(tx), reserva); err != nil {
			return err
		}
		// No permitir iniciar una reserva de otro día ni una atención sin personal activo.
		var valido bool
		err = tx.QueryRowContext(ctx, `SELECT r.fecha_reserva=(now() AT TIME ZONE 'America/Lima')::date
   AND EXISTS(SELECT 1 FROM asignaciones_trabajadores x JOIN trabajadores t ON t.id_usuario=x.id_trabajador
   JOIN usuarios u ON u.id_usuario=t.id_usuario WHERE x.id_atencion=$1 AND u.activo AND t.fecha_cese IS NULL)
   FROM reservas r WHERE r.id_reserva=$2`, id, reserva).Scan(&valido)
		if err != nil {
			return err
		}
		if !valido {
			return utils.Conflict("la atención debe estar asignada y programada para hoy")
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE atenciones SET estado=$2::estado_atencion,
 fecha_inicio_real=CASE WHEN $2='en_proceso' THEN now() ELSE fecha_inicio_real END,
 fecha_fin_real=CASE WHEN $2='finalizada' THEN now() ELSE fecha_fin_real END WHERE id_atencion=$1`, id, nuevo)
	if err != nil {
		return err
	}
	if nuevo == Finalizada {
		if _, err = tx.ExecContext(ctx, `UPDATE reservas SET estado='completada' WHERE id_reserva=$1`, reserva); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO historial_estados_atencion(id_historial,id_atencion,estado,registrado_por,comentario)
 VALUES($1,$2,$3::estado_atencion,$4,$5)`, uuid.New(), id, nuevo, c.UserID, "Estado actualizado desde la atención")
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Rendimiento es la única fórmula compartida por RF-13 y los futuros reportes RF-14.
func (repo *repository) Rendimiento(ctx context.Context, sede, id uuid.UUID, rango utils.RangoFechas) (*Rendimiento, error) {
	var existe bool
	err := repo.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM trabajadores t JOIN usuarios u ON u.id_usuario=t.id_usuario WHERE t.id_usuario=$1 AND u.id_sede=$2)`, id, sede).Scan(&existe)
	if err != nil {
		return nil, err
	}
	if !existe {
		return nil, utils.NotFound("trabajador no encontrado")
	}
	filas, err := ConsultarRendimientoEquipo(ctx, repo.db, sede, &id, rango)
	if err != nil {
		return nil, err
	}
	if len(filas) == 0 {
		return &Rendimiento{}, nil
	}
	return &Rendimiento{
		Atenciones:       filas[0].Atenciones,
		Servicios:        filas[0].Servicios,
		DuracionPromedio: filas[0].DuracionPromedio,
	}, nil
}
