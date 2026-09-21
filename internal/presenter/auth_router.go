package presenter

import (
	"io/fs"
	"net/http"
	"strings"

	"quiz-system/web"
)

// RegisterAuthRoutes configures public authentication, language switcher, and static asset routes
func RegisterAuthRoutes(mux *http.ServeMux, authPres *AuthPresenter, authLimiter *IPRateLimiter) {
	// Static Assets served from embedded filesystem
	staticSubFS, err := fs.Sub(web.StaticFS, "static")
	var fileServer http.Handler
	if err == nil {
		fileServer = http.FileServer(http.FS(staticSubFS))
	} else {
		fileServer = http.FileServer(http.Dir("./web/static"))
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Root redirect
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	// Language Switcher Endpoint
	mux.HandleFunc("GET /set-lang", func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")
		if lang != "ar" && lang != "en" {
			lang = "en"
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "lang_pref",
			Value:    lang,
			Path:     "/",
			MaxAge:   365 * 24 * 3600,
			Secure:   IsHTTPS(r),
			SameSite: http.SameSiteLaxMode,
		})
		redirectURL := r.URL.Query().Get("redirect")
		// F-17: Prevent open redirect – only allow relative paths starting with /
		if redirectURL == "" || !strings.HasPrefix(redirectURL, "/") || strings.HasPrefix(redirectURL, "//") {
			redirectURL = "/login"
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	})

	// Public Auth Endpoints
	mux.HandleFunc("GET /login", authPres.RenderLogin)
	mux.Handle("POST /login", authLimiter.LimitMiddleware(http.HandlerFunc(authPres.HandleLogin)))
	mux.HandleFunc("POST /logout", authPres.HandleLogout)
}
