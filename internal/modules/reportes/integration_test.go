package reportes_test

import (
	"testing"

	"carwashinmaribackend/internal/database"
	"carwashinmaribackend/internal/modules/reportes"
	"carwashinmaribackend/internal/testutil"
	"carwashinmaribackend/internal/utils"
	"github.com/google/uuid"
)

func TestDashboardReconciliaPagosOperacionYProductividad(t *testing.T) {
	db := testutil.Database(t)
	if err := database.ApplyMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	f := testutil.Seed(t, db)
	testutil.Exec(t, db, `UPDATE reservas SET estado='completada',fecha_reserva='2026-09-10' WHERE id_reserva=$1`, f.Reserva)
	testutil.Exec(t, db, `UPDATE atenciones SET estado='finalizada',fecha_inicio_real='2026-09-10 10:00:00-05',fecha_fin_real='2026-09-10 11:00:00-05' WHERE id_atencion=$1`, f.Atencion)
	// El duplicado deliberado comprueba que la productividad cuenta una sola
	// participación por trabajador y atención.
	testutil.Exec(t, db, `INSERT INTO asignaciones_trabajadores(id_atencion,id_trabajador) VALUES($1,$2)`, f.Atencion, f.Worker.UserID)

	testutil.Exec(t, db, `INSERT INTO pagos(id_pago,id_atencion,monto,metodo,estado,comprobante_interno,registrado_por,fecha_pago,
 moneda,idempotency_key,request_hash,fecha_reversion,motivo_reversion,revertido_por)
 VALUES($1,$2,20,'efectivo','reembolsado','CI-RF14-ANTERIOR',$3,'2026-08-31 12:00:00-05',
 'PEN','rf14-anterior','hash-anterior','2026-09-10 12:00:00-05','Prueba entre periodos',$3)`, uuid.New(), f.Atencion, f.Admin.UserID)
	testutil.Exec(t, db, `INSERT INTO pagos(id_pago,id_atencion,monto,metodo,estado,comprobante_interno,registrado_por,fecha_pago,
 moneda,idempotency_key,request_hash)
 VALUES($1,$2,20,'tarjeta','pagado','CI-RF14-ACTUAL',$3,'2026-09-11 12:00:00-05','PEN','rf14-actual','hash-actual')`, uuid.New(), f.Atencion, f.Admin.UserID)

	rango, err := utils.ParseRangoFechas("2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err := reportes.NewService(reportes.NewRepository(db)).Consultar(t.Context(), f.Admin.SedeID, rango)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Ingresos.Brutos != "20.00" || dashboard.Ingresos.Reembolsados != "20.00" || dashboard.Ingresos.Netos != "0.00" {
		t.Fatalf("reconciliación financiera incorrecta: %+v", dashboard.Ingresos)
	}
	if len(dashboard.Ingresos.PorTipo) != 1 || dashboard.Ingresos.PorTipo[0].Bruto != "20.00" || dashboard.Ingresos.PorTipo[0].Reembolso != "20.00" {
		t.Fatalf("desglose financiero incorrecto: %+v", dashboard.Ingresos.PorTipo)
	}
	if len(dashboard.Servicios) != 2 || dashboard.Servicios[0].Cantidad+dashboard.Servicios[1].Cantidad != 3 {
		t.Fatalf("servicios incorrectos: %+v", dashboard.Servicios)
	}
	if len(dashboard.Productividad) != 2 {
		t.Fatalf("equipo incorrecto: %+v", dashboard.Productividad)
	}
	for _, fila := range dashboard.Productividad {
		if fila.Atenciones != 1 || fila.Servicios != 3 || fila.DuracionPromedio == nil || *fila.DuracionPromedio != 60 {
			t.Fatalf("productividad duplicada o incorrecta: %+v", fila)
		}
	}
	if len(dashboard.Vehiculos) != 1 || dashboard.Vehiculos[0].Tipo != "Sin clasificar" || dashboard.Vehiculos[0].Atenciones != 1 {
		t.Fatalf("tipos de vehículo incorrectos: %+v", dashboard.Vehiculos)
	}
	if dashboard.Reservas.Total != 1 || dashboard.Reservas.Canceladas != 0 {
		t.Fatalf("reservas incorrectas: %+v", dashboard.Reservas)
	}
}

func TestDashboardPeriodoVacioYOtraSede(t *testing.T) {
	db := testutil.Database(t)
	if err := database.ApplyMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	f := testutil.Seed(t, db)
	rango, err := utils.ParseRangoFechas("2025-01-01", "2025-01-31")
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err := reportes.NewService(reportes.NewRepository(db)).Consultar(t.Context(), f.OtroAdmin.SedeID, rango)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Ingresos.Netos != "0" || dashboard.Reservas.Total != 0 || len(dashboard.Productividad) != 0 || len(dashboard.Vehiculos) != 0 {
		t.Fatalf("el periodo vacío expuso datos: %+v", dashboard)
	}
}
