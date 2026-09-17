package middleware

import (
	"context"
	"net/http"
	"strings"

	"carwashinmaribackend/internal/utils"
)

type contextKey string

const UserContextKey = contextKey("userClaims")

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			utils.ErrorJSON(w, http.StatusUnauthorized, "Token ausente o formato inválido")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := utils.ValidarJWT(tokenStr)
		if err != nil {
			utils.ErrorJSON(w, http.StatusUnauthorized, "Token inválido o expirado")
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRoles(rolesPermitidos ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*utils.CustomClaims)
			if !ok {
				utils.ErrorJSON(w, http.StatusForbidden, "Acceso denegado")
				return
			}

			permitido := false
			for _, rol := range rolesPermitidos {
				if strings.EqualFold(claims.Rol, rol) {
					permitido = true
					break
				}
			}

			if !permitido {
				utils.ErrorJSON(w, http.StatusForbidden, "No tienes permisos para esta acción")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
