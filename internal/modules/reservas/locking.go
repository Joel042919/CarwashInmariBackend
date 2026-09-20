package reservas

import (
	"carwashinmaribackend/internal/utils"
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
)

// Comparte el orden reserva -> atención con las transiciones del módulo atenciones.
func bloquearReservaModificable(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	var estado string
	err := tx.QueryRowContext(ctx, `SELECT estado::text FROM reservas WHERE id_reserva=$1 FOR UPDATE`, id).Scan(&estado)
	if errors.Is(err, sql.ErrNoRows) {
		return utils.NotFound("reserva no encontrada")
	}
	if err != nil {
		return err
	}
	if estado == EstadoCancelada || estado == EstadoCompletada {
		return utils.Conflict("la reserva ya no se puede modificar")
	}
	var estadoAtencion string
	err = tx.QueryRowContext(ctx, `SELECT estado::text FROM atenciones WHERE id_reserva=$1 FOR UPDATE`, id).Scan(&estadoAtencion)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if estadoAtencion != "programada" {
		return utils.Conflict("la atención ya comenzó; no se pueden cambiar sus asignaciones ni cancelar la reserva")
	}
	return nil
}
