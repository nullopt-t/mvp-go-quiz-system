package presenter

import (
	"context"
	"net/http"
	"strings"

	"quiz-system/internal/model"
	"quiz-system/internal/service"
)

type contextKey string

const (
	AuthClaimsKey contextKey = "auth_claims"
)

func AuthMiddleware(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := ""

			// 1. Check HTTP-only cookie first
			cookie, err := r.Cookie("jwt_token")
			if err == nil && cookie != nil && cookie.Value != "" {
				tokenString = cookie.Value
			}

			// 2. Check Authorization Header as fallback
			if tokenString == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if tokenString == "" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				// Expired or invalid token: clear cookie and redirect
				http.SetCookie(w, &http.Cookie{
					Name:     "jwt_token",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				http.Redirect(w, r, "/login?error=Session+expired.+Please+log+in+again.", http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), AuthClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(AuthClaimsKey).(*model.AuthClaims)
			if !ok || claims == nil || claims.Role != role {
				http.Error(w, "Forbidden: Unauthorized access", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetAuthClaims(r *http.Request) *model.AuthClaims {
	if claims, ok := r.Context().Value(AuthClaimsKey).(*model.AuthClaims); ok {
		return claims
	}
	return nil
}
