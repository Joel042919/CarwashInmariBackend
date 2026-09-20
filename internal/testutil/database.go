package testutil

import (
	"carwashinmaribackend/internal/utils"
	"database/sql"
	_ "embed"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net/url"
	"os"
	"strings"
	"testing"
)

//go:embed testdata/schema.sql
var schema string

// Database crea una base desechable local; nunca usa DATABASE_URL ni Neon.
func Database(t *testing.T) *sql.DB {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL no configurada; requiere PostgreSQL local para integración")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal("TEST_DATABASE_URL inválida")
	}
	if u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" {
		t.Fatal("las pruebas solo admiten PostgreSQL local")
	}
	admin, err := sql.Open("pgx", raw)
	if err != nil {
		t.Fatal(err)
	}
	name := "inmari_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(`CREATE DATABASE ` + name); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := admin.Exec(`DROP DATABASE ` + name + ` WITH (FORCE)`)
		if err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	u.Path = "/" + name
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	lines := []string{}
	for _, line := range strings.Split(schema, "\n") {
		if !strings.HasPrefix(line, `\`) {
			lines = append(lines, line)
		}
	}
	ddl := strings.ReplaceAll(strings.Join(lines, "\n"), "CREATE SCHEMA public;", "CREATE SCHEMA IF NOT EXISTS public;")
	// Una sola conexión aplica los SET de pg_dump; luego se restablece search_path.
	conn, err := db.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = conn.ExecContext(t.Context(), ddl); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	if _, err = conn.ExecContext(t.Context(), `SET search_path=public`); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	return db
}

func Exec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}

type Fixture struct {
	Admin, Worker, Worker2, Cliente, OtroAdmin utils.CustomClaims
	Reserva, Atencion, Servicio, Espacio       uuid.UUID
}

func Seed(t *testing.T, db *sql.DB) Fixture {
	t.Helper()
	sede, otra := uuid.New(), uuid.New()
	f := Fixture{
		Admin:     utils.CustomClaims{UserID: uuid.New(), SedeID: sede, Rol: "administrador"},
		Worker:    utils.CustomClaims{UserID: uuid.New(), SedeID: sede, Rol: "trabajador"},
		Worker2:   utils.CustomClaims{UserID: uuid.New(), SedeID: sede, Rol: "trabajador"},
		Cliente:   utils.CustomClaims{UserID: uuid.New(), SedeID: sede, Rol: "cliente"},
		OtroAdmin: utils.CustomClaims{UserID: uuid.New(), SedeID: otra, Rol: "administrador"},
		Reserva:   uuid.New(), Atencion: uuid.New(), Servicio: uuid.New(), Espacio: uuid.New(),
	}
	Exec(t, db, `INSERT INTO sede(id,sede_numero) VALUES($1,1),($2,2)`, sede, otra)
	roles := map[string]uuid.UUID{}
	for _, rol := range []string{"administrador", "trabajador", "cliente"} {
		id := uuid.New()
		roles[rol] = id
		Exec(t, db, `INSERT INTO rol(id,rol) VALUES($1,$2)`, id, rol)
	}
	for i, c := range []utils.CustomClaims{f.Admin, f.Worker, f.Worker2, f.Cliente, f.OtroAdmin} {
		Exec(t, db, `INSERT INTO usuarios(id_usuario,id_sede,nombre,apellido,correo,contrasena_hash,id_rol) VALUES($1,$2,'Prueba','RF13',$3,'hash-de-prueba',$4)`, c.UserID, c.SedeID, c.UserID.String()+"@example.test", roles[c.Rol])
		if c.Rol == "trabajador" {
			Exec(t, db, `INSERT INTO trabajadores(id_usuario,dni,fecha_contratacion) VALUES($1,$2,'2026-01-01')`, c.UserID, "TEST000"+string(rune('0'+i)))
		}
	}
	Exec(t, db, `INSERT INTO clientes(id_usuario) VALUES($1)`, f.Cliente.UserID)
	vehiculo, cat, servicio2 := uuid.New(), uuid.New(), uuid.New()
	Exec(t, db, `INSERT INTO vehiculos(id_vehiculo,id_cliente,placa,marca,modelo) VALUES($1,$2,'TEST-13','Prueba','RF13')`, vehiculo, f.Cliente.UserID)
	Exec(t, db, `INSERT INTO espacios_lavado(id_espacio,codigo) VALUES($1,'PRUEBA')`, f.Espacio)
	Exec(t, db, `INSERT INTO categoria_servicios(id_categoria_servicio,categoria_servicio) VALUES($1,'Prueba')`, cat)
	Exec(t, db, `INSERT INTO servicios(id_servicio,nombre,precio,duracion_estimada_min,id_categoria) VALUES($1,'Lavado',10,30,$3),($2,'Aspirado',5,15,$3)`, f.Servicio, servicio2, cat)
	Exec(t, db, `INSERT INTO reservas(id_reserva,id_cliente,id_vehiculo,id_espacio,fecha_reserva,hora_inicio,hora_fin,estado,total_estimado)
 VALUES($1,$2,$3,$4,(now() AT TIME ZONE 'America/Lima')::date,'08:00','09:00','confirmada',20)`, f.Reserva, f.Cliente.UserID, vehiculo, f.Espacio)
	Exec(t, db, `INSERT INTO reserva_servicios(id_reserva,id_servicio,precio_unitario,cantidad) VALUES($1,$2,10,1),($1,$3,5,2)`, f.Reserva, f.Servicio, servicio2)
	Exec(t, db, `INSERT INTO atenciones(id_atencion,id_reserva) VALUES($1,$2)`, f.Atencion, f.Reserva)
	Exec(t, db, `INSERT INTO asignaciones_trabajadores(id_atencion,id_trabajador) VALUES($1,$2),($1,$3)`, f.Atencion, f.Worker.UserID, f.Worker2.UserID)
	Exec(t, db, `INSERT INTO historial_estados_atencion(id_atencion,estado,registrado_por) VALUES($1,'programada',$2)`, f.Atencion, f.Admin.UserID)
	return f
}
