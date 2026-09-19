package middleware

import (
	"context"

	"carwashinmaribackend/internal/utils"
)

// ClaimsFromContext devuelve los claims JWT puestos por AuthMiddleware.
func ClaimsFromContext(ctx context.Context) (*utils.CustomClaims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*utils.CustomClaims)
	return claims, ok
}
