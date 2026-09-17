package presenter

import (
	"net/http"

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

	// 1. Register Public & Shared Auth Routes
	RegisterAuthRoutes(mux, deps.AuthPresenter)

	// Protected Auth Middleware
	authMiddleware := AuthMiddleware(deps.AuthService)

	// 2. Register Student-Specific Routes
	RegisterStudentRoutes(mux, authMiddleware, deps.StudentPres, deps.QuizPresenter)

	// 3. Register Admin / Staff-Specific Routes
	RegisterAdminRoutes(mux, authMiddleware, deps.AdminPresenter)

	// 4. Apply Global Internationalization & Language Switcher Middleware
	return I18nMiddleware()(mux)
}
