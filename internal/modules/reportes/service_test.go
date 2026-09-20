package reportes

import (
	"context"
	"testing"
	"time"

	"carwashinmaribackend/internal/utils"
	"github.com/google/uuid"
)

func TestNormalizarRangoPredeterminado(t *testing.T) {
	rango, err := normalizarRango(utils.RangoFechas{}, time.Date(2026, 9, 19, 18, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if got := rango.Desde.In(utils.ZonaNegocio).Format("2006-01-02"); got != "2026-08-21" {
		t.Fatalf("inicio predeterminado: %s", got)
	}
	if got := rango.Hasta.In(utils.ZonaNegocio).Format("2006-01-02"); got != "2026-09-20" {
		t.Fatalf("fin exclusivo predeterminado: %s", got)
	}
}

func TestNormalizarRangoRechazaMasDeUnAnio(t *testing.T) {
	desde := time.Date(2025, 1, 1, 0, 0, 0, 0, utils.ZonaNegocio)
	hasta := desde.AddDate(1, 0, 2)
	if _, err := normalizarRango(utils.RangoFechas{Desde: &desde, Hasta: &hasta}, time.Now()); err == nil {
		t.Fatal("se aceptó un periodo mayor a 366 días")
	}
}

// Verifica en compilación que el repositorio del caso de uso mantiene un
// contrato pequeño y no necesita conocer al usuario completo.
type repositorioNulo struct{}

func (repositorioNulo) Consultar(context.Context, uuid.UUID, utils.RangoFechas) (*Dashboard, error) {
	return &Dashboard{}, nil
}

var _ Repository = repositorioNulo{}
