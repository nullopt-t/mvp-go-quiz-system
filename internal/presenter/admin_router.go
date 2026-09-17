package presenter

import (
	"net/http"
	"strings"
)

// RegisterAdminRoutes configures routes specific to staff and administrator users
func RegisterAdminRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	authPres *AuthPresenter,
	adminPres *AdminPresenter,
) {
	// 1. Dedicated Admin Authentication Endpoints (Hidden from student login)
	mux.HandleFunc("GET /admin/login", authPres.RenderAdminLogin)
	mux.HandleFunc("POST /admin/login", authPres.HandleAdminLogin)
	mux.HandleFunc("POST /admin/logout", authPres.HandleAdminLogout)

	// 2. Guard Protected Admin Routes
	adminAuth := func(h http.Handler) http.Handler {
		return authMiddleware(RequireRole("admin")(h))
	}

	// Admin Overview & Dashboard
	mux.Handle("GET /admin", adminAuth(http.HandlerFunc(adminPres.RenderDashboard)))

	// Student Management & CSV Import
	mux.Handle("GET /admin/students", adminAuth(http.HandlerFunc(adminPres.RenderStudentsList)))
	mux.Handle("GET /admin/students/create", adminAuth(http.HandlerFunc(adminPres.RenderCreateStudent)))
	mux.Handle("POST /admin/students/create", adminAuth(http.HandlerFunc(adminPres.HandleCreateStudent)))
	mux.Handle("POST /admin/students/import", adminAuth(http.HandlerFunc(adminPres.HandleImportStudents)))
	mux.Handle("GET /admin/students/sample-csv", adminAuth(http.HandlerFunc(adminPres.HandleDownloadSampleCSV)))
	mux.Handle("GET /admin/students/", adminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/admin/students/")
		if path != "" && path != "create" && path != "sample-csv" && path != "import" {
			adminPres.RenderStudentProfile(w, r)
			return
		}
		adminPres.RenderStudentsList(w, r)
	})))

	// Quiz Authoring
	mux.Handle("GET /admin/quizzes/create", adminAuth(http.HandlerFunc(adminPres.RenderCreateQuiz)))
	mux.Handle("POST /admin/quizzes/create", adminAuth(http.HandlerFunc(adminPres.HandleCreateQuiz)))

	// Analytics & Leaderboards
	mux.Handle("GET /admin/quizzes/", adminAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/analytics") {
			adminPres.RenderQuizAnalytics(w, r)
			return
		}
		http.NotFound(w, r)
	})))
}
