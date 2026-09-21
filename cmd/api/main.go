package main

import (
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/modules/atenciones"
	"carwashinmaribackend/internal/modules/auth"
	"carwashinmaribackend/internal/modules/clientes"
	"carwashinmaribackend/internal/modules/documentos"
	"carwashinmaribackend/internal/modules/espacios"
	"carwashinmaribackend/internal/modules/pagos"
	"carwashinmaribackend/internal/modules/pedidos"
	"carwashinmaribackend/internal/modules/productos"
	"carwashinmaribackend/internal/modules/reclamos"
	"carwashinmaribackend/internal/modules/reportes"
	"carwashinmaribackend/internal/modules/reservas"
	"carwashinmaribackend/internal/modules/servicios"
	"carwashinmaribackend/internal/modules/trabajadores"
	"carwashinmaribackend/internal/modules/vehiculos"
	"carwashinmaribackend/internal/utils"
)

func main() {
	_ = godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Error de conexión a BD: %v", err)
	}
	defer db.Close()

	r := chi.NewRouter()

	// Middlewares globales requeridos por React Native / PWA
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	// Inicializar cliente R2 (con fallback local si no está disponible)
	r2Client, err := utils.NewR2Client(context.Background())
	if err != nil {
		log.Printf("[ADVERTENCIA] No se pudo inicializar Cloudflare R2: %v. Se usará almacenamiento local.", err)
	} else if r2Client == nil {
		log.Println("[INFO] Cloudflare R2 no configurado en .env. Se usará almacenamiento local en ./uploads.")
	} else {
		log.Println("[OK] Cloudflare R2 conectado correctamente.")
	}

	// Asegurar directorio local para uploads / evidencias
	_ = os.MkdirAll(filepath.Join(".", "uploads", "evidencias"), 0755)

	// Servir archivos estáticos locales de uploads
	workDir, _ := os.Getwd()
	uploadsDir := http.Dir(filepath.Join(workDir, "uploads"))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(uploadsDir)))

	// Inicialización repositorios y servicios (JOEL)
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo)
	authHandler := auth.NewHandler(authService)

	clientesRepo := clientes.NewRepository(db)
	clientesService := clientes.NewService(clientesRepo)
	clientesHandler := clientes.NewHandler(clientesService)

	reclamosRepo := reclamos.NewRepository(db)
	reclamosService := reclamos.NewService(reclamosRepo)
	reclamosHandler := reclamos.NewHandler(reclamosService, r2Client)

	// API Routes
	r.Route("/api/v1", func(api chi.Router) {
		// Rutas públicas
		api.Post("/auth/login", authHandler.Login)
		api.Post("/auth/registro-cliente", authHandler.RegistrarCliente)

		// Servir evidencias desde R2 o fallback local con Content-Type y Cache (GET y HEAD)
		archivosHandler := func(w http.ResponseWriter, req *http.Request) {
			key := chi.URLParam(req, "*")
			if key == "" {
				http.NotFound(w, req)
				return
			}

			// Intento 1: Cloudflare R2
			if r2Client != nil {
				body, cType, err := r2Client.ObtenerArchivo(req.Context(), key)
				if err == nil {
					defer body.Close()
					w.Header().Set("Content-Type", cType)
					w.Header().Set("Cache-Control", "public, max-age=86400")
					if req.Method != http.MethodHead {
						_, _ = io.Copy(w, body)
					}
					return
				}
			}

			// Intento 2: Archivo en uploads local
			localPath := filepath.Join(".", "uploads", filepath.Clean(key))
			if _, err := os.Stat(localPath); err == nil {
				http.ServeFile(w, req, localPath)
				return
			}

			http.NotFound(w, req)
		}
		api.Get("/archivos/*", archivosHandler)
		api.Head("/archivos/*", archivosHandler)

		// Rutas protegidas por JWT
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.AuthMiddleware)
			protected.Use(middleware.ActiveUser(db))

			// RF-02: Clientes
			protected.Get("/clientes/perfil", clientesHandler.ObtenerMiPerfil)
			protected.Put("/clientes/perfil", clientesHandler.ActualizarMiPerfil)
			protected.Get("/clientes/historial", clientesHandler.ObtenerHistorialCompleto)

			// RF-15: Reclamos (Clientes y Admin)
			protected.Post("/reclamos", reclamosHandler.CrearReclamo)
			protected.Get("/reclamos/mis-reclamos", reclamosHandler.ListarMisReclamos)

			// Solo Administrador
			protected.Group(func(admin chi.Router) {
				admin.Use(middleware.RequireRoles("administrador"))
				admin.Get("/admin/reclamos", reclamosHandler.ListarTodos)
				admin.Patch("/admin/reclamos/{id}/responder", reclamosHandler.ResponderReclamo)
			})

			// -------------------------------------------------------------
			// AQUÍ CONECTAN LOS DEMÁS:
			// -------------------------------------------------------------
			// Fatima (RF-04, RF-07, RF-12): terminado.
			servicios.RegisterRoutes(protected, db)
			documentos.RegisterRoutes(protected, db, r2Client)
			productos.RegisterRoutes(protected, db)
			pedidos.RegisterRoutes(protected, db)
			pagos.RegisterRoutes(protected, db)

			// RF-05 / RF-06 / RF-09 (Erick): espacios, horarios, reservas y programación.
			espacios.RegisterRoutes(protected, db)
			reservas.RegisterRoutes(protected, db, documentos.NewChecker(db))
			// Mego (RF-03, RF-08, RF-10): solo hay un registro mínimo de vehículos.
			//   TODO(Mego): RF-03 completo, RF-08 evidencias y RF-10 trazabilidad.
			vehiculos.RegisterRoutes(protected, db)
			// Ingrid (RF-11, RF-13, RF-14): pagos, personal, atenciones e indicadores.
			trabajadores.RegisterRoutes(protected, db)
			atenciones.RegisterRoutes(protected, db)
			reportes.RegisterRoutes(protected, db)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor corriendo en :%s", port)
	http.ListenAndServe(":"+port, r)
}
