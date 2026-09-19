package reservas

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

// HorarioEspacio es un tramo de atención de un espacio activo en un día concreto.
type HorarioEspacio struct {
	IDEspacio uuid.UUID
	Codigo    string
	Tramo     Rango
}

type Repository interface {
	// Lecturas para validar
	VehiculoDeCliente(ctx context.Context, idVehiculo, idCliente uuid.UUID) (bool, error)
	EspacioActivo(ctx context.Context, idEspacio uuid.UUID) (bool, error)
	TramosEspacioDia(ctx context.Context, idEspacio uuid.UUID, dia int) ([]Rango, error)
	HorariosActivosDia(ctx context.Context, dia int) ([]HorarioEspacio, error)
	OcupacionDia(ctx context.Context, fecha string, excluir *uuid.UUID) (map[uuid.UUID][]Rango, error)
	ServiciosPorID(ctx context.Context, ids []uuid.UUID) ([]ServicioInfo, error)

	// Escrituras (transaccionales)
	CrearReserva(ctx context.Context, r *Reserva, servicios []ServicioInfo) error
	ReprogramarReserva(ctx context.Context, id uuid.UUID, fecha, inicio, fin string, idEspacio uuid.UUID) error
	CancelarReserva(ctx context.Context, id uuid.UUID, motivo string) error
	ProgramarReserva(ctx context.Context, id, idAdmin uuid.UUID, trabajadores []uuid.UUID) error

	// Consultas
	ObtenerReserva(ctx context.Context, id uuid.UUID) (*Reserva, error)
	ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]Reserva, error)
	ListarTodas(ctx context.Context, estado, fecha string) ([]Reserva, error)
	TrabajadoresParaReserva(ctx context.Context, r *Reserva) ([]TrabajadorDisponible, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ---------------------------------------------------------------- lecturas

func (r *repository) existe(ctx context.Context, query string, args ...any) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&ok)
	return ok, err
}

func (r *repository) VehiculoDeCliente(ctx context.Context, idVehiculo, idCliente uuid.UUID) (bool, error) {
	return r.existe(ctx,
		`SELECT EXISTS(SELECT 1 FROM vehiculos WHERE id_vehiculo = $1 AND id_cliente = $2)`, idVehiculo, idCliente)
}

func (r *repository) EspacioActivo(ctx context.Context, idEspacio uuid.UUID) (bool, error) {
	return r.existe(ctx,
		`SELECT EXISTS(SELECT 1 FROM espacios_lavado WHERE id_espacio = $1 AND activo)`, idEspacio)
}

func leerRango(ini, fin string) (Rango, error) {
	a, err := utils.ParseHM(ini)
	if err != nil {
		return Rango{}, err
	}
	b, err := utils.ParseHM(fin)
	if err != nil {
		return Rango{}, err
	}
	return Rango{Inicio: a, Fin: b}, nil
}

func (r *repository) TramosEspacioDia(ctx context.Context, idEspacio uuid.UUID, dia int) ([]Rango, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT to_char(hora_inicio, 'HH24:MI'), to_char(hora_fin, 'HH24:MI')
		FROM horarios_atencion
		WHERE id_espacio = $1 AND dia_semana = $2 AND hora_fin IS NOT NULL
		ORDER BY hora_inicio`, idEspacio, dia)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tramos []Rango
	for rows.Next() {
		var ini, fin string
		if err := rows.Scan(&ini, &fin); err != nil {
			return nil, err
		}
		t, err := leerRango(ini, fin)
		if err != nil {
			return nil, err
		}
		tramos = append(tramos, t)
	}
	return tramos, rows.Err()
}

func (r *repository) HorariosActivosDia(ctx context.Context, dia int) ([]HorarioEspacio, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT e.id_espacio, e.codigo, to_char(h.hora_inicio, 'HH24:MI'), to_char(h.hora_fin, 'HH24:MI')
		FROM horarios_atencion h
		INNER JOIN espacios_lavado e ON e.id_espacio = h.id_espacio
		WHERE e.activo AND h.dia_semana = $1 AND h.hora_fin IS NOT NULL
		ORDER BY e.codigo, h.hora_inicio`, dia)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []HorarioEspacio
	for rows.Next() {
		var h HorarioEspacio
		var ini, fin string
		if err := rows.Scan(&h.IDEspacio, &h.Codigo, &ini, &fin); err != nil {
			return nil, err
		}
		t, err := leerRango(ini, fin)
		if err != nil {
			return nil, err
		}
		h.Tramo = t
		lista = append(lista, h)
	}
	return lista, rows.Err()
}

func (r *repository) OcupacionDia(ctx context.Context, fecha string, excluir *uuid.UUID) (map[uuid.UUID][]Rango, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id_espacio, to_char(hora_inicio, 'HH24:MI'), to_char(hora_fin, 'HH24:MI')
		FROM reservas
		WHERE fecha_reserva = $1::date AND estado::text <> 'cancelada'
		  AND ($2::uuid IS NULL OR id_reserva <> $2::uuid)`, fecha, excluir)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ocup := map[uuid.UUID][]Rango{}
	for rows.Next() {
		var id uuid.UUID
		var ini, fin string
		if err := rows.Scan(&id, &ini, &fin); err != nil {
			return nil, err
		}
		t, err := leerRango(ini, fin)
		if err != nil {
			return nil, err
		}
		ocup[id] = append(ocup[id], t)
	}
	return ocup, rows.Err()
}

func (r *repository) ServiciosPorID(ctx context.Context, ids []uuid.UUID) ([]ServicioInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	marcas := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		marcas[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id_servicio, nombre, precio, duracion_estimada_min, disponible, requiere_documento
		FROM servicios WHERE id_servicio IN (`+strings.Join(marcas, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []ServicioInfo
	for rows.Next() {
		var s ServicioInfo
		if err := rows.Scan(&s.ID, &s.Nombre, &s.Precio, &s.DuracionMin, &s.Disponible, &s.RequiereDocumento); err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, rows.Err()
}

// ---------------------------------------------------------------- escrituras

// bloquear serializa las operaciones concurrentes sobre la misma clave hasta el fin de la transacción.
func bloquear(ctx context.Context, tx *sql.Tx, clave string) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, clave)
	return err
}

func hayCruceEspacio(ctx context.Context, q queryRower, idEspacio uuid.UUID, fecha, inicio, fin string, excluir *uuid.UUID) (bool, error) {
	var cruce bool
	err := q.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM reservas
			WHERE id_espacio = $1 AND fecha_reserva = $2::date AND estado::text <> 'cancelada'
			  AND hora_inicio < $4::time AND hora_fin > $3::time
			  AND ($5::uuid IS NULL OR id_reserva <> $5::uuid))`,
		idEspacio, fecha, inicio, fin, excluir).Scan(&cruce)
	return cruce, err
}

var errHorarioOcupado = utils.Conflict("ese horario ya no está disponible, elige otro")

func (r *repository) CrearReserva(ctx context.Context, res *Reserva, servicios []ServicioInfo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := bloquear(ctx, tx, "espacio:"+res.IDEspacio.String()+":"+res.FechaReserva); err != nil {
		return err
	}
	cruce, err := hayCruceEspacio(ctx, tx, res.IDEspacio, res.FechaReserva, res.HoraInicio, res.HoraFin, nil)
	if err != nil {
		return err
	}
	if cruce {
		return errHorarioOcupado
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO reservas (
			id_reserva, id_cliente, id_vehiculo, id_espacio, fecha_reserva, hora_inicio, hora_fin,
			estado, total_estimado, observaciones, fecha_creacion
		) VALUES ($1, $2, $3, $4, $5::date, $6::time, $7::time, $8, $9, $10, $11)`,
		res.IDReserva, res.IDCliente, res.IDVehiculo, res.IDEspacio, res.FechaReserva,
		res.HoraInicio, res.HoraFin, res.Estado, res.TotalEstimado, res.Observaciones, res.FechaCreacion); err != nil {
		return err
	}

	for _, s := range servicios {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reserva_servicios (id_reserva_servicio, id_reserva, id_servicio, precio_unitario, cantidad)
			VALUES ($1, $2, $3, $4, 1)`,
			uuid.New(), res.IDReserva, s.ID, s.Precio); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *repository) ReprogramarReserva(ctx context.Context, id uuid.UUID, fecha, inicio, fin string, idEspacio uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := bloquear(ctx, tx, "espacio:"+idEspacio.String()+":"+fecha); err != nil {
		return err
	}
	cruce, err := hayCruceEspacio(ctx, tx, idEspacio, fecha, inicio, fin, &id)
	if err != nil {
		return err
	}
	if cruce {
		return errHorarioOcupado
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE reservas
		SET fecha_reserva = $2::date, hora_inicio = $3::time, hora_fin = $4::time,
			id_espacio = $5, estado = $6
		WHERE id_reserva = $1`,
		id, fecha, inicio, fin, idEspacio, EstadoReprogramada)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// borrarAtencion elimina la atención "programada" de una reserva (con su historial y asignaciones).
func borrarAtencion(ctx context.Context, tx *sql.Tx, idReserva uuid.UUID) error {
	var idAtencion uuid.UUID
	err := tx.QueryRowContext(ctx, `SELECT id_atencion FROM atenciones WHERE id_reserva = $1`, idReserva).Scan(&idAtencion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, q := range []string{
		`DELETE FROM asignaciones_trabajadores WHERE id_atencion = $1`,
		`DELETE FROM historial_estados_atencion WHERE id_atencion = $1`,
		`DELETE FROM atenciones WHERE id_atencion = $1`,
	} {
		if _, err := tx.ExecContext(ctx, q, idAtencion); err != nil {
			return err
		}
	}
	return nil
}

func (r *repository) CancelarReserva(ctx context.Context, id uuid.UUID, motivo string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := borrarAtencion(ctx, tx, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE reservas SET estado = $2, motivo_cancelacion = $3, id_trabajador = NULL
		WHERE id_reserva = $1`, id, EstadoCancelada, motivo); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *repository) ProgramarReserva(ctx context.Context, id, idAdmin uuid.UUID, trabajadores []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var estado, fecha, inicio, fin string
	err = tx.QueryRowContext(ctx, `
		SELECT estado::text, fecha_reserva::text, to_char(hora_inicio, 'HH24:MI'), to_char(hora_fin, 'HH24:MI')
		FROM reservas WHERE id_reserva = $1 FOR UPDATE`, id).Scan(&estado, &fecha, &inicio, &fin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return utils.NotFound("reserva no encontrada")
		}
		return err
	}
	if estado == EstadoCancelada || estado == EstadoCompletada {
		return utils.Conflict("no se puede programar una reserva " + estado)
	}

	for _, idTrab := range trabajadores {
		if err := bloquear(ctx, tx, "trabajador:"+idTrab.String()+":"+fecha); err != nil {
			return err
		}

		var disponible bool
		err := tx.QueryRowContext(ctx, `
			SELECT t.disponible FROM trabajadores t
			INNER JOIN usuarios u ON u.id_usuario = t.id_usuario
			WHERE t.id_usuario = $1 AND u.activo AND t.fecha_cese IS NULL`, idTrab).Scan(&disponible)
		if errors.Is(err, sql.ErrNoRows) {
			return utils.BadRequest("uno de los trabajadores no existe o está dado de baja")
		}
		if err != nil {
			return err
		}
		if !disponible {
			return utils.BadRequest("uno de los trabajadores está marcado como no disponible")
		}

		var ocupado bool
		if err := tx.QueryRowContext(ctx, sqlTrabajadorOcupado, idTrab, id, fecha, inicio, fin).Scan(&ocupado); err != nil {
			return err
		}
		if ocupado {
			return utils.Conflict("un trabajador ya tiene otra atención que se cruza con este horario")
		}
	}

	// Atención: se crea la primera vez y se reutiliza si la reserva se reprograma.
	var idAtencion uuid.UUID
	err = tx.QueryRowContext(ctx, `SELECT id_atencion FROM atenciones WHERE id_reserva = $1`, id).Scan(&idAtencion)
	if errors.Is(err, sql.ErrNoRows) {
		idAtencion = uuid.New()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO atenciones (id_atencion, id_reserva, estado) VALUES ($1, $2, 'programada')`,
			idAtencion, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO historial_estados_atencion (id_historial, id_atencion, estado, registrado_por, comentario)
			VALUES ($1, $2, 'programada', $3, 'Atención programada por el administrador')`,
			uuid.New(), idAtencion, idAdmin); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM asignaciones_trabajadores WHERE id_atencion = $1`, idAtencion); err != nil {
		return err
	}
	for _, idTrab := range trabajadores {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO asignaciones_trabajadores (id_asignacion, id_atencion, id_trabajador)
			VALUES ($1, $2, $3)`, uuid.New(), idAtencion, idTrab); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE reservas SET estado = $2, id_trabajador = $3 WHERE id_reserva = $1`,
		id, EstadoConfirmada, trabajadores[0]); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------------------------------------------------------------- consultas

const selectReserva = `
	SELECT r.id_reserva, r.id_cliente, u.nombre || ' ' || u.apellido, u.correo,
		r.id_vehiculo, v.placa, v.marca || ' ' || v.modelo,
		r.id_espacio, e.codigo,
		r.fecha_reserva::text, to_char(r.hora_inicio, 'HH24:MI'), to_char(r.hora_fin, 'HH24:MI'),
		r.estado::text, r.total_estimado, r.observaciones, r.motivo_cancelacion, r.fecha_creacion,
		a.id_atencion, a.estado::text
	FROM reservas r
	INNER JOIN usuarios u ON u.id_usuario = r.id_cliente
	INNER JOIN vehiculos v ON v.id_vehiculo = r.id_vehiculo
	INNER JOIN espacios_lavado e ON e.id_espacio = r.id_espacio
	LEFT JOIN atenciones a ON a.id_reserva = r.id_reserva
`

type scanner interface {
	Scan(dest ...any) error
}

func scanReserva(sc scanner, x *Reserva) error {
	return sc.Scan(
		&x.IDReserva, &x.IDCliente, &x.NombreCliente, &x.CorreoCliente,
		&x.IDVehiculo, &x.Placa, &x.Vehiculo,
		&x.IDEspacio, &x.CodigoEspacio,
		&x.FechaReserva, &x.HoraInicio, &x.HoraFin,
		&x.Estado, &x.TotalEstimado, &x.Observaciones, &x.MotivoCancelacion, &x.FechaCreacion,
		&x.IDAtencion, &x.EstadoAtencion,
	)
}

func (r *repository) cargarDetalle(ctx context.Context, x *Reserva) error {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id_servicio, s.nombre, rs.precio_unitario, rs.cantidad, s.duracion_estimada_min, s.requiere_documento
		FROM reserva_servicios rs
		INNER JOIN servicios s ON s.id_servicio = rs.id_servicio
		WHERE rs.id_reserva = $1 ORDER BY s.nombre`, x.IDReserva)
	if err != nil {
		return err
	}
	x.Servicios = []ReservaServicio{}
	for rows.Next() {
		var s ReservaServicio
		if err := rows.Scan(&s.IDServicio, &s.Nombre, &s.PrecioUnitario, &s.Cantidad, &s.DuracionMin, &s.RequiereDocumento); err != nil {
			rows.Close()
			return err
		}
		if s.RequiereDocumento {
			x.RequiereDocumento = true
		}
		x.Servicios = append(x.Servicios, s)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	rows, err = r.db.QueryContext(ctx, `
		SELECT a.id_trabajador, u.nombre || ' ' || u.apellido
		FROM asignaciones_trabajadores a
		INNER JOIN atenciones att ON att.id_atencion = a.id_atencion
		INNER JOIN usuarios u ON u.id_usuario = a.id_trabajador
		WHERE att.id_reserva = $1 ORDER BY u.nombre`, x.IDReserva)
	if err != nil {
		return err
	}
	defer rows.Close()
	x.Trabajadores = []TrabajadorAsignado{}
	for rows.Next() {
		var t TrabajadorAsignado
		if err := rows.Scan(&t.IDTrabajador, &t.Nombre); err != nil {
			return err
		}
		x.Trabajadores = append(x.Trabajadores, t)
	}
	return rows.Err()
}

func (r *repository) listar(ctx context.Context, query string, args ...any) ([]Reserva, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	lista := []Reserva{}
	for rows.Next() {
		var x Reserva
		if err := scanReserva(rows, &x); err != nil {
			rows.Close()
			return nil, err
		}
		lista = append(lista, x)
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

func (r *repository) ObtenerReserva(ctx context.Context, id uuid.UUID) (*Reserva, error) {
	var x Reserva
	if err := scanReserva(r.db.QueryRowContext(ctx, selectReserva+` WHERE r.id_reserva = $1`, id), &x); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := r.cargarDetalle(ctx, &x); err != nil {
		return nil, err
	}
	return &x, nil
}

func (r *repository) ListarPorCliente(ctx context.Context, idCliente uuid.UUID) ([]Reserva, error) {
	return r.listar(ctx, selectReserva+` WHERE r.id_cliente = $1
		ORDER BY r.fecha_reserva DESC, r.hora_inicio DESC`, idCliente)
}

func (r *repository) ListarTodas(ctx context.Context, estado, fecha string) ([]Reserva, error) {
	return r.listar(ctx, selectReserva+`
		WHERE ($1 = '' OR r.estado::text = $1)
		  AND ($2 = '' OR r.fecha_reserva = $2::date)
		ORDER BY r.fecha_reserva DESC, r.hora_inicio DESC`, estado, fecha)
}

// sqlTrabajadorOcupado: parámetros idTrabajador, idReservaExcluida, fecha, inicio, fin.
const sqlTrabajadorOcupado = `
	SELECT EXISTS(
		SELECT 1
		FROM asignaciones_trabajadores a
		INNER JOIN atenciones att ON att.id_atencion = a.id_atencion
		INNER JOIN reservas r ON r.id_reserva = att.id_reserva
		WHERE a.id_trabajador = $1 AND r.id_reserva <> $2
		  AND r.fecha_reserva = $3::date AND r.estado::text <> 'cancelada'
		  AND r.hora_inicio < $5::time AND r.hora_fin > $4::time)`

func (r *repository) TrabajadoresParaReserva(ctx context.Context, x *Reserva) ([]TrabajadorDisponible, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id_usuario, u.nombre || ' ' || u.apellido, t.disponible,
			EXISTS(
				SELECT 1
				FROM asignaciones_trabajadores a
				INNER JOIN atenciones att ON att.id_atencion = a.id_atencion
				INNER JOIN reservas r ON r.id_reserva = att.id_reserva
				WHERE a.id_trabajador = t.id_usuario AND r.id_reserva <> $1
				  AND r.fecha_reserva = $2::date AND r.estado::text <> 'cancelada'
				  AND r.hora_inicio < $4::time AND r.hora_fin > $3::time
			) AS ocupado
		FROM trabajadores t
		INNER JOIN usuarios u ON u.id_usuario = t.id_usuario
		WHERE u.activo AND t.fecha_cese IS NULL
		ORDER BY u.nombre, u.apellido`,
		x.IDReserva, x.FechaReserva, x.HoraInicio, x.HoraFin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lista := []TrabajadorDisponible{}
	for rows.Next() {
		var t TrabajadorDisponible
		if err := rows.Scan(&t.IDTrabajador, &t.Nombre, &t.Disponible, &t.Ocupado); err != nil {
			return nil, err
		}
		lista = append(lista, t)
	}
	return lista, rows.Err()
}
