package atenciones_test

import (
	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/modules/atenciones"
	"carwashinmaribackend/internal/modules/reservas"
	"carwashinmaribackend/internal/modules/trabajadores"
	"carwashinmaribackend/internal/testutil"
	"carwashinmaribackend/internal/utils"
	"context"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func status(t *testing.T, err error, want int) {
	t.Helper()
	var ae *utils.AppError
	if !errors.As(err, &ae) || ae.Status != want {
		t.Fatalf("esperaba HTTP %d; recibió %v", want, err)
	}
}

func TestCicloCompletoYPermisos(t *testing.T) {
	db := testutil.Database(t)
	f := testutil.Seed(t, db)
	ctx := context.Background()
	svc := atenciones.NewService(atenciones.NewRepository(db))
	personal := trabajadores.NewService(trabajadores.NewRepository(db))
	status(t, personal.DarBaja(ctx, f.Admin.SedeID, f.Worker.UserID), 409)
	lista, err := svc.Listar(ctx, f.Worker, atenciones.Filtro{})
	if err != nil || len(lista) != 1 {
		t.Fatalf("asignaciones: %v %v", lista, err)
	}
	lista, err = svc.Listar(ctx, f.OtroAdmin, atenciones.Filtro{})
	if err != nil || len(lista) != 0 {
		t.Fatalf("filtrado de sede: %v", err)
	}
	ajeno := f.Worker
	ajeno.UserID = uuid.New()
	status(t, svc.CambiarEstado(ctx, ajeno, f.Atencion, "en_proceso"), 404)
	status(t, svc.CambiarEstado(ctx, f.Cliente, f.Atencion, "en_proceso"), 403)
	status(t, svc.CambiarEstado(ctx, f.Worker, f.Atencion, "finalizada"), 409)
	testutil.Exec(t, db, `UPDATE servicios SET requiere_documento=true WHERE id_servicio=$1`, f.Servicio)
	status(t, svc.CambiarEstado(ctx, f.Worker, f.Atencion, "en_proceso"), 409)
	testutil.Exec(t, db, `INSERT INTO documentos_previos(id_reserva,id_servicio,id_cliente,ruta_pdf,estado) VALUES($1,$2,$3,'prueba.pdf','validado')`, f.Reserva, f.Servicio, f.Cliente.UserID)
	if err = svc.CambiarEstado(ctx, f.Worker, f.Atencion, "en_proceso"); err != nil {
		t.Fatal(err)
	}
	repoReservas := reservas.NewRepository(db)
	status(t, repoReservas.ProgramarReserva(ctx, f.Reserva, f.Admin.UserID, []uuid.UUID{f.Worker2.UserID}), 409)
	status(t, repoReservas.CancelarReserva(ctx, f.Reserva, "prueba"), 409)
	status(t, repoReservas.ReprogramarReserva(ctx, f.Reserva, "2026-12-01", "08:00", "09:00", f.Espacio), 409)
	status(t, personal.DarBaja(ctx, f.Admin.SedeID, f.Worker.UserID), 409)
	var wg sync.WaitGroup
	resultados := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); resultados <- svc.CambiarEstado(ctx, f.Worker, f.Atencion, "finalizada") }()
	}
	wg.Wait()
	close(resultados)
	exitos := 0
	for err := range resultados {
		if err == nil {
			exitos++
		} else {
			status(t, err, 409)
		}
	}
	if exitos != 1 {
		t.Fatalf("finalizaciones: %d", exitos)
	}
	eventos, err := svc.Historial(ctx, f.Cliente, f.Atencion)
	if err != nil || len(eventos) != 3 {
		t.Fatalf("historial: %v %v", eventos, err)
	}
	eventos, err = svc.Historial(ctx, f.OtroAdmin, f.Atencion)
	if err != nil || len(eventos) != 0 {
		t.Fatal("historial ajeno expuesto")
	}
	testutil.Exec(t, db, `UPDATE atenciones SET fecha_inicio_real=fecha_fin_real-interval '60 minutes' WHERE id_atencion=$1`, f.Atencion)
	for _, id := range []uuid.UUID{f.Worker.UserID, f.Worker2.UserID} {
		metricas, err := svc.Rendimiento(ctx, f.Admin, id, utils.RangoFechas{})
		if err != nil {
			t.Fatal(err)
		}
		if metricas.Atenciones != 1 || metricas.Servicios != 3 || metricas.DuracionPromedio == nil || *metricas.DuracionPromedio != 60 {
			t.Fatalf("métricas duplicadas o incorrectas: %+v", metricas)
		}
	}
	manana := time.Now().In(utils.ZonaNegocio).AddDate(0, 0, 1).Format("2006-01-02")
	rango, _ := utils.ParseRangoFechas(manana, manana)
	metricas, err := svc.Rendimiento(ctx, f.Admin, f.Worker.UserID, rango)
	if err != nil || metricas.Atenciones != 0 || metricas.DuracionPromedio != nil {
		t.Fatalf("rango vacío: %+v %v", metricas, err)
	}
	t.Setenv("JWT_SECRET", "secreto-exclusivo-de-pruebas-rf13")
	token, err := utils.GenerarJWT(f.Worker.UserID, f.Worker.SedeID, "trabajador")
	if err != nil {
		t.Fatal(err)
	}
	h := middleware.AuthMiddleware(middleware.ActiveUser(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })))
	request := func() int {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}
	if code := request(); code != 204 {
		t.Fatalf("sesión activa: %d", code)
	}
	if err = personal.DarBaja(ctx, f.Admin.SedeID, f.Worker.UserID); err != nil {
		t.Fatal(err)
	}
	if code := request(); code != 401 {
		t.Fatalf("token de baja permitido: %d", code)
	}
	metricas, err = svc.Rendimiento(ctx, f.Admin, f.Worker.UserID, utils.RangoFechas{})
	if err != nil || metricas.Atenciones != 1 {
		t.Fatal("baja eliminó historial")
	}
}

func TestTrabajadoresEdicionYRollback(t *testing.T) {
	db := testutil.Database(t)
	f := testutil.Seed(t, db)
	ctx := context.Background()
	svc := trabajadores.NewService(trabajadores.NewRepository(db))
	in := trabajadores.CrearTrabajadorInput{Nombre: " Nueva ", Apellido: "Persona", Correo: "NUEVA@example.test", DNI: "12345678", Contrasena: "prueba-123"}
	nuevo, err := svc.Crear(ctx, f.Admin.SedeID, in)
	if err != nil {
		t.Fatal(err)
	}
	if nuevo.Nombre != "Nueva" || nuevo.Correo != "nueva@example.test" || !nuevo.Activo {
		t.Fatalf("normalización: %+v", nuevo)
	}
	_, err = svc.Crear(ctx, f.Admin.SedeID, in)
	status(t, err, 409)
	editar := trabajadores.ActualizarTrabajadorInput{Nombre: "Editada", Apellido: "Persona", Correo: "editada@example.test", DNI: "87654321", FechaContratacion: "2026-01-02"}
	_, err = svc.Actualizar(ctx, f.OtroAdmin.SedeID, nuevo.IDUsuario, editar)
	status(t, err, 404)
	actualizado, err := svc.Actualizar(ctx, f.Admin.SedeID, nuevo.IDUsuario, editar)
	if err != nil || actualizado.Nombre != "Editada" {
		t.Fatal(err)
	}
	editar.Nombre = "No persistir"
	editar.DNI = "TEST0001" // DNI existente: falla después de UPDATE usuarios.
	_, err = svc.Actualizar(ctx, f.Admin.SedeID, nuevo.IDUsuario, editar)
	status(t, err, 409)
	var nombre string
	if err = db.QueryRow(`SELECT nombre FROM usuarios WHERE id_usuario=$1`, nuevo.IDUsuario).Scan(&nombre); err != nil || nombre != "Editada" {
		t.Fatalf("rollback falló: %s %v", nombre, err)
	}
	if err = svc.DarBaja(ctx, f.Admin.SedeID, nuevo.IDUsuario); err != nil {
		t.Fatal(err)
	}
	status(t, svc.CambiarDisponibilidad(ctx, f.Admin.SedeID, nuevo.IDUsuario, true), 404)
	listado, err := svc.Listar(ctx, f.OtroAdmin.SedeID)
	if err != nil || len(listado) != 0 {
		t.Fatal("trabajadores de otra sede expuestos")
	}
}

func TestReprogramacionLiberaAsignacionYExigeConfirmar(t *testing.T) {
	db := testutil.Database(t)
	f := testutil.Seed(t, db)
	ctx := context.Background()
	repo := reservas.NewRepository(db)
	fecha := time.Now().In(utils.ZonaNegocio).Format("2006-01-02")
	if err := repo.ReprogramarReserva(ctx, f.Reserva, fecha, "10:00", "11:00", f.Espacio); err != nil {
		t.Fatal(err)
	}
	var cantidad int
	db.QueryRow(`SELECT count(*) FROM asignaciones_trabajadores WHERE id_atencion=$1`, f.Atencion).Scan(&cantidad)
	if cantidad != 0 {
		t.Fatal("conservó personal sin validar el nuevo horario")
	}
	svc := atenciones.NewService(atenciones.NewRepository(db))
	status(t, svc.CambiarEstado(ctx, f.Admin, f.Atencion, "en_proceso"), 409)
	if err := repo.ProgramarReserva(ctx, f.Reserva, f.Admin.UserID, []uuid.UUID{f.Worker.UserID}); err != nil {
		t.Fatal(err)
	}
	if err := svc.CambiarEstado(ctx, f.Worker, f.Atencion, "en_proceso"); err != nil {
		t.Fatal(err)
	}
}
