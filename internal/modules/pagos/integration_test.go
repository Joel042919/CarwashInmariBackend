package pagos_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"carwashinmaribackend/internal/database"
	"carwashinmaribackend/internal/modules/pagos"
	"carwashinmaribackend/internal/testutil"
	"carwashinmaribackend/internal/utils"
	"github.com/google/uuid"
)

func appStatus(t *testing.T, err error, want int) {
	t.Helper()
	var appErr *utils.AppError
	if !errors.As(err, &appErr) || appErr.Status != want {
		t.Fatalf("esperaba estado %d; recibió %v", want, err)
	}
}

func TestPagoAtencionIdempotenciaComprobanteYReversion(t *testing.T) {
	db := testutil.Database(t)
	if err := database.ApplyMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	f := testutil.Seed(t, db)
	testutil.Exec(t, db, `UPDATE atenciones SET estado='finalizada',fecha_inicio_real=now()-interval '1 hour',fecha_fin_real=now() WHERE id_atencion=$1`, f.Atencion)
	testutil.Exec(t, db, `UPDATE reservas SET estado='completada' WHERE id_reserva=$1`, f.Reserva)

	service := pagos.NewService(pagos.NewRepository(db))
	input := pagos.RegistrarPagoInput{
		Tipo: pagos.TipoAtencion, IDOperacion: f.Atencion, Metodo: "efectivo",
		IdempotencyKey: "rf11-idempotencia-001",
	}
	primero, err := service.Registrar(t.Context(), f.Admin.SedeID, f.Admin.UserID, input)
	if err != nil {
		t.Fatal(err)
	}
	if primero.Monto != "20.00" || primero.Estado != pagos.Pagado || primero.ComprobanteInterno == "" {
		t.Fatalf("pago incorrecto: %+v", primero)
	}
	repetido, err := service.Registrar(t.Context(), f.Admin.SedeID, f.Admin.UserID, input)
	if err != nil || repetido.IDPago != primero.IDPago {
		t.Fatalf("reintento no idempotente: %+v %v", repetido, err)
	}
	input.Metodo = "tarjeta"
	_, err = service.Registrar(t.Context(), f.Admin.SedeID, f.Admin.UserID, input)
	appStatus(t, err, 409)
	input.IdempotencyKey = "rf11-idempotencia-002"
	_, err = service.Registrar(t.Context(), f.Admin.SedeID, f.Admin.UserID, input)
	appStatus(t, err, 409)

	comprobante, err := service.ObtenerComprobante(t.Context(), f.Cliente, primero.IDPago)
	if err != nil || comprobante.IDPago != primero.IDPago {
		t.Fatalf("comprobante del cliente: %+v %v", comprobante, err)
	}
	ajeno := f.Cliente
	ajeno.UserID = uuid.New()
	_, err = service.ObtenerComprobante(t.Context(), ajeno, primero.IDPago)
	appStatus(t, err, 404)
	_, err = service.Revertir(t.Context(), f.Admin.SedeID, f.Admin.UserID, primero.IDPago, pagos.RevertirPagoInput{Motivo: "Corrección solicitada", ReembolsoConfirmado: false})
	appStatus(t, err, 400)
	revertido, err := service.Revertir(t.Context(), f.Admin.SedeID, f.Admin.UserID, primero.IDPago, pagos.RevertirPagoInput{Motivo: "Devolución realizada al cliente", ReembolsoConfirmado: true})
	if err != nil || revertido.Estado != pagos.Reembolsado || revertido.FechaReversion == nil {
		t.Fatalf("reversión incorrecta: %+v %v", revertido, err)
	}
	_, err = service.Revertir(t.Context(), f.Admin.SedeID, f.Admin.UserID, primero.IDPago, pagos.RevertirPagoInput{Motivo: "Segundo intento inválido", ReembolsoConfirmado: true})
	appStatus(t, err, 409)
	input.IdempotencyKey = "rf11-idempotencia-003"
	nuevo, err := service.Registrar(t.Context(), f.Admin.SedeID, f.Admin.UserID, input)
	if err != nil || nuevo.IDPago == primero.IDPago {
		t.Fatalf("corrección de cobro inválida: %+v %v", nuevo, err)
	}
}

func TestOperacionesPendientesYFiltros(t *testing.T) {
	db := testutil.Database(t)
	if err := database.ApplyMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	f := testutil.Seed(t, db)
	testutil.Exec(t, db, `UPDATE atenciones SET estado='finalizada',fecha_inicio_real=$2,fecha_fin_real=$3 WHERE id_atencion=$1`, f.Atencion, time.Now().Add(-time.Hour), time.Now())
	testutil.Exec(t, db, `UPDATE reservas SET estado='completada' WHERE id_reserva=$1`, f.Reserva)
	service := pagos.NewService(pagos.NewRepository(db))
	lista, err := service.ListarPendientes(t.Context(), f.Admin.SedeID, pagos.TipoAtencion)
	if err != nil || len(lista) != 1 || lista[0].Monto != "20.00" {
		t.Fatalf("pendientes incorrectos: %+v %v", lista, err)
	}
	lista, err = service.ListarPendientes(t.Context(), f.OtroAdmin.SedeID, "")
	if err != nil || len(lista) != 0 {
		t.Fatalf("operaciones de otra sede expuestas: %+v %v", lista, err)
	}
}

func TestCobroConcurrenteNoSeDuplica(t *testing.T) {
	db := testutil.Database(t)
	if err := database.ApplyMigrations(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	f := testutil.Seed(t, db)
	testutil.Exec(t, db, `UPDATE atenciones SET estado='finalizada',fecha_inicio_real=now()-interval '1 hour',fecha_fin_real=now() WHERE id_atencion=$1`, f.Atencion)
	testutil.Exec(t, db, `UPDATE reservas SET estado='completada' WHERE id_reserva=$1`, f.Reserva)
	service := pagos.NewService(pagos.NewRepository(db))
	input := pagos.RegistrarPagoInput{
		Tipo: pagos.TipoAtencion, IDOperacion: f.Atencion, Metodo: "efectivo",
		IdempotencyKey: "rf11-concurrencia-001",
	}
	type resultado struct {
		pago *pagos.Pago
		err  error
	}
	resultados := make(chan resultado, 2)
	var grupo sync.WaitGroup
	for range 2 {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			pago, err := service.Registrar(context.Background(), f.Admin.SedeID, f.Admin.UserID, input)
			resultados <- resultado{pago: pago, err: err}
		}()
	}
	grupo.Wait()
	close(resultados)
	var id uuid.UUID
	for resultado := range resultados {
		if resultado.err != nil {
			t.Fatal(resultado.err)
		}
		if id == uuid.Nil {
			id = resultado.pago.IDPago
		} else if id != resultado.pago.IDPago {
			t.Fatal("el mismo reintento creó dos pagos")
		}
	}
	var cantidad int
	if err := db.QueryRow(`SELECT count(*) FROM pagos WHERE id_atencion=$1 AND estado='pagado'`, f.Atencion).Scan(&cantidad); err != nil {
		t.Fatal(err)
	}
	if cantidad != 1 {
		t.Fatalf("se registraron %d pagos vigentes", cantidad)
	}
}
