package presenter

import (
	"net/http"
	"strings"

	"quiz-system/internal/service"
)

type RouterDependencies struct {
	AuthService    service.AuthService
	AuthPresenter  *AuthPresenter
	StudentPres    *StudentPresenter
	QuizPresenter  *QuizPresenter
	AdminPresenter *AdminPresenter
}

func NewRouter(deps RouterDependencies) http.Handler {
	mux := http.NewServeMux()

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
	mux.HandleFunc("GET /login", deps.AuthPresenter.RenderLogin)
	mux.HandleFunc("POST /login", deps.AuthPresenter.HandleLogin)
	mux.HandleFunc("POST /logout", deps.AuthPresenter.HandleLogout)

	// Protected Routes Wrapper
	authMiddleware := AuthMiddleware(deps.AuthService)

	// Student Protected Routes
	mux.Handle("GET /dashboard", authMiddleware(http.HandlerFunc(deps.StudentPres.RenderDashboard)))

	// Student Quiz Routes
	mux.Handle("GET /quizzes/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/result") {
			deps.QuizPresenter.RenderQuizResult(w, r)
			return
		}
		// Default to quiz room
		deps.QuizPresenter.RenderQuizRoom(w, r)
	})))

	mux.Handle("POST /quizzes/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/answers") {
			deps.QuizPresenter.HandleNextAnswer(w, r)
			return
		}
		if strings.HasSuffix(path, "/submit") {
			deps.QuizPresenter.HandleSubmitQuiz(w, r)
			return
		}
		http.NotFound(w, r)
	})))

	// Admin Routes (Role: "admin")
	adminAuth := func(h http.Handler) http.Handler {
		return authMiddleware(RequireRole("admin")(h))
	}

	mux.Handle("GET /admin", adminAuth(http.HandlerFunc(deps.AdminPresenter.RenderDashboard)))
	mux.Handle("GET /admin/quizzes/create", adminAuth(http.HandlerFunc(deps.AdminPresenter.RenderCreateQuiz)))
	mux.Handle("POST /admin/quizzes/create", adminAuth(http.HandlerFunc(deps.AdminPresenter.HandleCreateQuiz)))
	mux.Handle("GET /admin/quizzes/", adminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/analytics") {
			deps.AdminPresenter.RenderQuizAnalytics(w, r)
			return
		}
		http.NotFound(w, r)
	})))

	// Apply I18n Middleware globally
	return I18nMiddleware()(mux)
}
