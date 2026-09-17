package presenter

import (
	"net/http"
	"strings"
)

// RegisterAdminRoutes configures protected routes specific to staff and administrator users
func RegisterAdminRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	adminPres *AdminPresenter,
) {
	// Guard admin routes with admin role check
	adminAuth := func(h http.Handler) http.Handler {
		return authMiddleware(RequireRole("admin")(h))
	}

	// Admin Overview & Dashboard
	mux.Handle("GET /admin", adminAuth(http.HandlerFunc(adminPres.RenderDashboard)))

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
