package presenter

import (
	"net/http"
	"strings"
)

// RegisterStudentRoutes configures protected routes specific to student users
func RegisterStudentRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	studentPres *StudentPresenter,
	quizPres *QuizPresenter,
) {
	// Student Dashboard
	mux.Handle("GET /dashboard", authMiddleware(http.HandlerFunc(studentPres.RenderDashboard)))

	// Student Quiz Routes (Start, Live Room, Results)
	mux.Handle("GET /quizzes/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/result") {
			quizPres.RenderQuizResult(w, r)
			return
		}
		// Default to quiz room
		quizPres.RenderQuizRoom(w, r)
	})))

	// Student Answer submission & final quiz completion
	mux.Handle("POST /quizzes/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/answers") {
			quizPres.HandleNextAnswer(w, r)
			return
		}
		if strings.HasSuffix(path, "/submit") {
			quizPres.HandleSubmitQuiz(w, r)
			return
		}
		http.NotFound(w, r)
	})))
}
