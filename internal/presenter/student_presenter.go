package presenter

import (
	"html/template"
	"net/http"

	"quiz-system/internal/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StudentPresenter struct {
	quizService   service.QuizService
	resultService service.ResultService
	templates     *template.Template
}

func NewStudentPresenter(quizService service.QuizService, resultService service.ResultService, tmpl *template.Template) *StudentPresenter {
	return &StudentPresenter{
		quizService:   quizService,
		resultService: resultService,
		templates:     tmpl,
	}
}

func (p *StudentPresenter) RenderDashboard(w http.ResponseWriter, r *http.Request) {
	claims := GetAuthClaims(r)
	if claims == nil || claims.Role != "student" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	stuObjID, err := primitive.ObjectIDFromHex(claims.StudentID)
	if err != nil {
		http.Error(w, "Invalid student identifier", http.StatusBadRequest)
		return
	}

	quizzes, err := p.quizService.GetAvailableQuizzesForStudent(r.Context(), stuObjID, claims.LevelID, claims.GroupID)
	if err != nil {
		http.Error(w, "Failed to load quizzes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	history, err := p.resultService.GetStudentHistory(r.Context(), stuObjID)
	if err != nil {
		history = nil
	}

	data := map[string]interface{}{
		"Student": claims,
		"Quizzes": quizzes,
		"History": history,
	}

	_ = p.templates.ExecuteTemplate(w, "dashboard.html", data)
}
