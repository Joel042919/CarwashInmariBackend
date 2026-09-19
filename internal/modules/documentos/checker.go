package documentos

import (
	"context"
	"database/sql"
	"strings"

	"github.com/google/uuid"

	"carwashinmaribackend/internal/utils"
)

// Checker es el punto de integración de RF-07 con el módulo de reservas.
//
// Regla de RF-07: una reserva con servicios que tengan requiere_documento = true no puede
// pasar a "confirmada" hasta que cada uno tenga un PDF del cliente en estado "validado".
// El módulo de reservas llama a ExigirDocumentosValidados antes de confirmar (ver Programar
// en reservas/service.go); cualquier otro flujo que confirme reservas debe hacer lo mismo.
type Checker interface {
	RequisitosReserva(ctx context.Context, idReserva uuid.UUID, idCliente *uuid.UUID) (*RequisitosReserva, error)
}

// NewChecker crea un Checker sobre la conexión a la base de datos.
func NewChecker(db *sql.DB) Checker {
	return NewService(NewRepository(db))
}

// ExigirDocumentosValidados devuelve nil si la reserva puede confirmarse, es decir,
// si todos sus servicios que exigen documento tienen un PDF en estado "validado".
// En caso contrario devuelve un error 409 que nombra los servicios pendientes, listo
// para responderse con utils.WriteError.
func ExigirDocumentosValidados(ctx context.Context, c Checker, idReserva uuid.UUID) error {
	res, err := c.RequisitosReserva(ctx, idReserva, nil)
	if err != nil {
		return err
	}
	if res.PuedeConfirmarse {
		return nil
	}

	pendientes := make([]string, 0, len(res.Requisitos))
	for _, r := range res.Requisitos {
		if !r.Cumplido {
			pendientes = append(pendientes, r.NombreServicio)
		}
	}
	return utils.Conflict("la reserva no puede confirmarse: falta documento validado para " + strings.Join(pendientes, ", "))
}
