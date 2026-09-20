package reservas

import (
	"carwashinmaribackend/internal/middleware"
	"carwashinmaribackend/internal/utils"
	"database/sql"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
)

func reservaDeSede(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := chi.URLParam(r, "id")
			if raw == "" {
				next.ServeHTTP(w, r)
				return
			}
			id, err := uuid.Parse(raw)
			if err != nil {
				utils.ErrorJSON(w, 400, "ID de reserva inválido")
				return
			}
			c, ok := middleware.ClaimsFromContext(r.Context())
			if !ok {
				utils.ErrorJSON(w, 401, "Sesión requerida")
				return
			}
			var existe bool
			err = db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM reservas r JOIN usuarios u ON u.id_usuario=r.id_cliente WHERE r.id_reserva=$1 AND u.id_sede=$2)`, id, c.SedeID).Scan(&existe)
			if err != nil {
				utils.WriteError(w, err)
				return
			}
			if !existe {
				utils.ErrorJSON(w, 404, "reserva no encontrada")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
