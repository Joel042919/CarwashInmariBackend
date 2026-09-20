package middleware

import (
	"carwashinmaribackend/internal/utils"
	"database/sql"
	"net/http"
)

// ActiveUser revoca efectivamente el acceso al dar de baja una cuenta.
// También rechaza tokens con rol o sede que ya no correspondan al usuario.
func ActiveUser(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, ok := ClaimsFromContext(r.Context())
			if !ok {
				utils.ErrorJSON(w, 401, "Sesión requerida")
				return
			}
			var valido bool
			err := db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM usuarios u JOIN rol r ON r.id=u.id_rol
   WHERE u.id_usuario=$1 AND u.id_sede=$2 AND lower(r.rol)=lower($3) AND u.activo
   AND (lower(r.rol)<>'trabajador' OR EXISTS(SELECT 1 FROM trabajadores t WHERE t.id_usuario=u.id_usuario AND t.fecha_cese IS NULL)))`, c.UserID, c.SedeID, c.Rol).Scan(&valido)
			if err != nil {
				utils.WriteError(w, err)
				return
			}
			if !valido {
				utils.ErrorJSON(w, 401, "Tu cuenta está inactiva o tu sesión ya no es válida")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
