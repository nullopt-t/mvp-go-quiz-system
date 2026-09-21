package presenter

import (
	"context"
	"net/http"
	"strings"

	"quiz-system/internal/i18n"
	"quiz-system/internal/model"
	"quiz-system/internal/service"
)

type contextKey string

const (
	AuthClaimsKey contextKey = "auth_claims"
	I18nBundleKey contextKey = "i18n_bundle"
)

func I18nMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if language was explicitly requested in URL
			if qLang := r.URL.Query().Get("lang"); qLang != "" {
				http.SetCookie(w, &http.Cookie{
					Name:     "lang_pref",
					Value:    qLang,
					Path:     "/",
					MaxAge:   365 * 24 * 3600,
					Secure:   IsHTTPS(r),
					SameSite: http.SameSiteLaxMode,
				})
			}

			lang := i18n.GetLanguage(r)
			bundle := i18n.GetBundle(lang)

			ctx := context.WithValue(r.Context(), I18nBundleKey, bundle)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetI18n(r *http.Request) i18n.TranslationBundle {
	if bundle, ok := r.Context().Value(I18nBundleKey).(i18n.TranslationBundle); ok {
		return bundle
	}
	return i18n.GetBundle(i18n.LangEN)
}

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

			loginPath := "/login"
			if strings.HasPrefix(r.URL.Path, "/admin") {
				loginPath = "/admin/login"
			}

			if tokenString == "" {
				http.Redirect(w, r, loginPath, http.StatusSeeOther)
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
					Secure:   IsHTTPS(r),
				})
				SetFlashError(w, "Session expired. Please log in again.")
				http.Redirect(w, r, loginPath, http.StatusSeeOther)
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
