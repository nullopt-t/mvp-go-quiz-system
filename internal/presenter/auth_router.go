package presenter

import (
	"net/http"
)

// RegisterAuthRoutes configures public authentication, language switcher, and static asset routes
func RegisterAuthRoutes(mux *http.ServeMux, authPres *AuthPresenter) {
	// Static Assets
	fileServer := http.FileServer(http.Dir("./web/static"))
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
			SameSite: http.SameSiteLaxMode,
		})
		redirectURL := r.URL.Query().Get("redirect")
		if redirectURL == "" {
			redirectURL = "/login"
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	})

	// Public Auth Endpoints
	mux.HandleFunc("GET /login", authPres.RenderLogin)
	mux.HandleFunc("POST /login", authPres.HandleLogin)
	mux.HandleFunc("POST /logout", authPres.HandleLogout)
}
