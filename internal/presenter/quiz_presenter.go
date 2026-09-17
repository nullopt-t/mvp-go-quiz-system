package presenter

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"quiz-system/internal/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type QuizPresenter struct {
	quizService   service.QuizService
	answerService service.AnswerService
	resultService service.ResultService
	templates     *template.Template
}

func NewQuizPresenter(
	quizService service.QuizService,
	answerService service.AnswerService,
	resultService service.ResultService,
	tmpl *template.Template,
) *QuizPresenter {
	return &QuizPresenter{
		quizService:   quizService,
		answerService: answerService,
		resultService: resultService,
		templates:     tmpl,
	}
}

func (p *QuizPresenter) RenderQuizRoom(w http.ResponseWriter, r *http.Request) {
	claims := GetAuthClaims(r)
	if claims == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	quizIDHex := strings.TrimPrefix(r.URL.Path, "/quizzes/")
	quizIDHex = strings.TrimSuffix(quizIDHex, "/start")
	quizIDHex = strings.Split(quizIDHex, "/")[0]

	quizObjID, err := primitive.ObjectIDFromHex(quizIDHex)
	if err != nil {
		http.Error(w, "Invalid Quiz ID", http.StatusBadRequest)
		return
	}

	stuObjID, err := primitive.ObjectIDFromHex(claims.StudentID)
	if err != nil {
		http.Error(w, "Invalid Student ID", http.StatusBadRequest)
		return
	}

	quizView, err := p.quizService.StartOrResumeQuiz(r.Context(), quizObjID, stuObjID)
	if err != nil {
		http.Error(w, "Failed to start quiz: "+err.Error(), http.StatusBadRequest)
		return
	}

	if quizView.IsCompleted {
		http.Redirect(w, r, fmt.Sprintf("/quizzes/%s/result", quizIDHex), http.StatusSeeOther)
		return
	}

	// Determine starting question index: first unanswered question
	startIndex := 0
	for i, q := range quizView.Quiz.Questions {
		if _, answered := quizView.PreviousAnswers[q.ID]; !answered {
			startIndex = i
			break
		}
	}

	currentQuestion := quizView.Quiz.Questions[startIndex]
	selectedOptionID := quizView.PreviousAnswers[currentQuestion.ID]

	i18nBundle := GetI18n(r)

	data := map[string]interface{}{
		"Student":          claims,
		"Quiz":             quizView.Quiz,
		"TotalQuestions":   len(quizView.Quiz.Questions),
		"CurrentIndex":     startIndex,
		"CurrentNumber":    startIndex + 1,
		"CurrentQuestion":  currentQuestion,
		"SelectedOptionID": selectedOptionID,
		"RemainingSeconds": quizView.RemainingSeconds,
		"IsLastQuestion":   startIndex == len(quizView.Quiz.Questions)-1,
		"PreviousAnswers":  quizView.PreviousAnswers,
		"I18n":             i18nBundle,
	}

	_ = p.templates.ExecuteTemplate(w, "quiz_room.html", data)
}

// HandleNextAnswer is triggered by the NEXT button via HTMX POST
func (p *QuizPresenter) HandleNextAnswer(w http.ResponseWriter, r *http.Request) {
	claims := GetAuthClaims(r)
	if claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form submission", http.StatusBadRequest)
		return
	}

	quizIDHex := r.FormValue("quiz_id")
	quizObjID, err := primitive.ObjectIDFromHex(quizIDHex)
	if err != nil {
		http.Error(w, "Invalid Quiz ID", http.StatusBadRequest)
		return
	}

	stuObjID, err := primitive.ObjectIDFromHex(claims.StudentID)
	if err != nil {
		http.Error(w, "Invalid Student ID", http.StatusBadRequest)
		return
	}

	questionID, _ := strconv.Atoi(r.FormValue("question_id"))
	answerID, _ := strconv.Atoi(r.FormValue("answer_id"))
	currentIndex, _ := strconv.Atoi(r.FormValue("current_index"))
	isFinalSubmit := r.FormValue("is_submit") == "true"

	// 1. Append Answer if selected (No SQL update, pure append-only insert)
	if answerID > 0 && questionID > 0 {
		_ = p.answerService.RecordAnswer(r.Context(), quizObjID, stuObjID, claims.StudentCode, questionID, answerID)
	}

	// 2. Fetch Quiz to render next question or finalize
	quiz, err := p.quizService.GetQuizByID(r.Context(), quizObjID)
	if err != nil || quiz == nil {
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}

	nextIndex := currentIndex + 1

	// If last question was submitted or finalize requested
	if isFinalSubmit || nextIndex >= len(quiz.Questions) {
		// Calculate final results
		_, _ = p.resultService.CalculateAndSubmit(
			r.Context(),
			quizObjID,
			stuObjID,
			claims.StudentCode,
			claims.Name,
			claims.LevelID,
			claims.GroupID,
		)

		// Instruct HTMX to redirect to results page
		w.Header().Set("HX-Redirect", fmt.Sprintf("/quizzes/%s/result", quizIDHex))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Retrieve updated answers state
	latestAnswers, _ := p.answerService.GetStudentAnswerState(r.Context(), quizObjID, stuObjID)
	nextQuestion := quiz.Questions[nextIndex]
	selectedOptionID := latestAnswers[nextQuestion.ID]

	i18nBundle := GetI18n(r)

	data := map[string]interface{}{
		"Student":          claims,
		"Quiz":             quiz,
		"TotalQuestions":   len(quiz.Questions),
		"CurrentIndex":     nextIndex,
		"CurrentNumber":    nextIndex + 1,
		"CurrentQuestion":  nextQuestion,
		"SelectedOptionID": selectedOptionID,
		"IsLastQuestion":   nextIndex == len(quiz.Questions)-1,
		"PreviousAnswers":  latestAnswers,
		"I18n":             i18nBundle,
	}

	// Render the next question partial dynamically via HTMX
	_ = p.templates.ExecuteTemplate(w, "question_card.html", data)
}

// HandleSubmitQuiz handles the final submission / timeout auto-submit
func (p *QuizPresenter) HandleSubmitQuiz(w http.ResponseWriter, r *http.Request) {
	claims := GetAuthClaims(r)
	if claims == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	quizIDHex := strings.TrimPrefix(r.URL.Path, "/quizzes/")
	quizIDHex = strings.TrimSuffix(quizIDHex, "/submit")

	quizObjID, err := primitive.ObjectIDFromHex(quizIDHex)
	if err != nil {
		http.Error(w, "Invalid Quiz ID", http.StatusBadRequest)
		return
	}

	stuObjID, err := primitive.ObjectIDFromHex(claims.StudentID)
	if err != nil {
		http.Error(w, "Invalid Student ID", http.StatusBadRequest)
		return
	}

	// Calculate and save results
	_, err = p.resultService.CalculateAndSubmit(
		r.Context(),
		quizObjID,
		stuObjID,
		claims.StudentCode,
		claims.Name,
		claims.LevelID,
		claims.GroupID,
	)
	if err != nil {
		http.Error(w, "Failed to calculate results: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/quizzes/%s/result", quizIDHex), http.StatusSeeOther)
}

// RenderQuizResult renders the student score report
func (p *QuizPresenter) RenderQuizResult(w http.ResponseWriter, r *http.Request) {
	claims := GetAuthClaims(r)
	if claims == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	quizIDHex := strings.TrimPrefix(r.URL.Path, "/quizzes/")
	quizIDHex = strings.TrimSuffix(quizIDHex, "/result")

	quizObjID, err := primitive.ObjectIDFromHex(quizIDHex)
	if err != nil {
		http.Error(w, "Invalid Quiz ID", http.StatusBadRequest)
		return
	}

	stuObjID, err := primitive.ObjectIDFromHex(claims.StudentID)
	if err != nil {
		http.Error(w, "Invalid Student ID", http.StatusBadRequest)
		return
	}

	quiz, err := p.quizService.GetQuizByID(r.Context(), quizObjID)
	if err != nil || quiz == nil {
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}

	result, err := p.resultService.GetStudentResult(r.Context(), quizObjID, stuObjID)
	if err != nil || result == nil {
		// Calculate if not yet calculated
		result, err = p.resultService.CalculateAndSubmit(
			r.Context(),
			quizObjID,
			stuObjID,
			claims.StudentCode,
			claims.Name,
			claims.LevelID,
			claims.GroupID,
		)
		if err != nil {
			http.Error(w, "Failed to retrieve result", http.StatusInternalServerError)
			return
		}
	}

	percentage := 0
	if result.TotalPoints > 0 {
		percentage = (result.Score * 100) / result.TotalPoints
	}

	i18nBundle := GetI18n(r)

	data := map[string]interface{}{
		"Student":    claims,
		"Quiz":       quiz,
		"Result":     result,
		"Percentage": percentage,
		"I18n":       i18nBundle,
	}

	_ = p.templates.ExecuteTemplate(w, "result.html", data)
}
