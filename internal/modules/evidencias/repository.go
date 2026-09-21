package evidencias

import (
	"carwashinmaribackend/internal/utils"
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
)

type Repository interface {
	Crear(context.Context, utils.CustomClaims, *Evidencia) error
	Listar(context.Context, utils.CustomClaims, uuid.UUID) ([]Evidencia, error)
}
type repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return &repository{db: db} }

const accesoAtencion = `u.id_sede=$1 AND (($2='administrador') OR ($2='cliente' AND r.id_cliente=$3) OR ($2='trabajador' AND EXISTS(SELECT 1 FROM asignaciones_trabajadores x WHERE x.id_atencion=a.id_atencion AND x.id_trabajador=$3)))`
const joinsAtencion = ` FROM atenciones a JOIN reservas r ON r.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=r.id_cliente `

func (r *repository) Crear(ctx context.Context, c utils.CustomClaims, e *Evidencia) error {
	var vehiculoReserva uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT r.id_vehiculo`+joinsAtencion+` WHERE `+accesoAtencion+` AND a.id_atencion=$4`, c.SedeID, c.Rol, c.UserID, e.IDAtencion).Scan(&vehiculoReserva)
	if errors.Is(err, sql.ErrNoRows) {
		return utils.NotFound("atención no encontrada o sin permiso")
	}
	if err != nil {
		return err
	}
	if vehiculoReserva != e.IDVehiculo {
		return utils.BadRequest("el vehículo no corresponde a esta atención")
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO evidencias(id_evidencia,id_atencion,registrado_por,id_vehiculo,tipo,descripcion,ruta_foto,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, e.IDEvidencia, e.IDAtencion, e.RegistradoPor, e.IDVehiculo, e.Tipo, e.Descripcion, e.RutaFoto, e.CreatedAt)
	return err
}

func (r *repository) Listar(ctx context.Context, c utils.CustomClaims, id uuid.UUID) ([]Evidencia, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT e.id_evidencia,e.id_atencion,e.id_vehiculo,e.registrado_por,e.tipo,e.descripcion,e.ruta_foto,e.created_at FROM evidencias e JOIN atenciones a ON a.id_atencion=e.id_atencion JOIN reservas r ON r.id_reserva=a.id_reserva JOIN usuarios u ON u.id_usuario=r.id_cliente WHERE `+accesoAtencion+` AND e.id_atencion=$4 ORDER BY e.created_at DESC`, c.SedeID, c.Rol, c.UserID, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Evidencia{}
	for rows.Next() {
		var e Evidencia
		if err := rows.Scan(&e.IDEvidencia, &e.IDAtencion, &e.IDVehiculo, &e.RegistradoPor, &e.Tipo, &e.Descripcion, &e.RutaFoto, &e.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}
