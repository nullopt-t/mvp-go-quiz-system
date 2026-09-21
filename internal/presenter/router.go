package presenter

import (
	"net/http"
	"time"

	"quiz-system/internal/service"
)

type RouterDependencies struct {
	AuthService    service.AuthService
	AuthPresenter  *AuthPresenter
	StudentPres    *StudentPresenter
	QuizPresenter  *QuizPresenter
	AdminPresenter *AdminPresenter
}

// NewRouter constructs and orchestrates modular sub-routers for each user role
func NewRouter(deps RouterDependencies) http.Handler {
	mux := http.NewServeMux()

	// Rate limiters:
	// - Student login: 5 req/s with burst of 10
	// - Admin login: 2 req/s with burst of 5 (strictly protects against PIN brute force)
	studentLoginLimiter := NewIPRateLimiter(5, 10, 10*time.Minute)
	adminLoginLimiter := NewIPRateLimiter(2, 5, 10*time.Minute)

	// 1. Register Public & Shared Auth Routes (Student Login at /login)
	RegisterAuthRoutes(mux, deps.AuthPresenter, studentLoginLimiter)

	// Protected Auth Middleware
	authMiddleware := AuthMiddleware(deps.AuthService)

	// 2. Register Student-Specific Routes (/quizzes, /quizzes/*)
	RegisterStudentRoutes(mux, authMiddleware, deps.StudentPres, deps.QuizPresenter)

	// 3. Register Admin / Staff-Specific Routes (/admin/login, /admin, /admin/quizzes/*)
	RegisterAdminRoutes(mux, authMiddleware, deps.AuthPresenter, deps.AdminPresenter, adminLoginLimiter)

	// 4. Wrap with Internationalization and Security Middleware
	handler := I18nMiddleware()(mux)
	return SecurityMiddleware()(handler)
}
