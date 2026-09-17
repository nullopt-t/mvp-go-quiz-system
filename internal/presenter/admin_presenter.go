package presenter

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"
	"quiz-system/internal/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AdminPresenter struct {
	quizService   service.QuizService
	resultService service.ResultService
	studentRepo   repository.StudentRepository
	templates     *template.Template
}

func NewAdminPresenter(
	quizService service.QuizService,
	resultService service.ResultService,
	studentRepo repository.StudentRepository,
	tmpl *template.Template,
) *AdminPresenter {
	return &AdminPresenter{
		quizService:   quizService,
		resultService: resultService,
		studentRepo:   studentRepo,
		templates:     tmpl,
	}
}

func (p *AdminPresenter) RenderDashboard(w http.ResponseWriter, r *http.Request) {
	quizzes, _ := p.quizService.GetAllQuizzes(r.Context())
	totalStudents, _ := p.studentRepo.Count(r.Context())

	// Summarize counts across 5 levels & 4 groups
	type GroupSummary struct {
		LevelID int
		GroupID int
		Count   int64
	}
	var groupSummaries []GroupSummary
	for l := 1; l <= 5; l++ {
		for g := 1; g <= 4; g++ {
			c, _ := p.studentRepo.CountByLevelAndGroup(r.Context(), l, g)
			groupSummaries = append(groupSummaries, GroupSummary{
				LevelID: l,
				GroupID: g,
				Count:   c,
			})
		}
	}

	data := map[string]interface{}{
		"Quizzes":        quizzes,
		"TotalStudents":  totalStudents,
		"GroupSummaries": groupSummaries,
	}

	_ = p.templates.ExecuteTemplate(w, "admin_dashboard.html", data)
}

func (p *AdminPresenter) RenderCreateQuiz(w http.ResponseWriter, r *http.Request) {
	_ = p.templates.ExecuteTemplate(w, "admin_create_quiz.html", nil)
}

func (p *AdminPresenter) HandleCreateQuiz(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	levelID, _ := strconv.Atoi(r.FormValue("level_id"))
	duration, _ := strconv.Atoi(r.FormValue("duration_minutes"))

	if duration <= 0 {
		duration = 20
	}

	// Group IDs parsing (e.g. "1,2,3")
	var groupIDs []int
	groupsInput := r.FormValue("group_ids")
	if groupsInput != "" {
		for _, part := range strings.Split(groupsInput, ",") {
			if g, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && g > 0 {
				groupIDs = append(groupIDs, g)
			}
		}
	}

	// Parse Dynamic Questions
	questionCount, _ := strconv.Atoi(r.FormValue("question_count"))
	if questionCount <= 0 {
		questionCount = 3
	}

	var questions []model.Question
	for qIdx := 1; qIdx <= questionCount; qIdx++ {
		qText := strings.TrimSpace(r.FormValue(fmt.Sprintf("q_%d_text", qIdx)))
		if qText == "" {
			continue
		}

		qPoints, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("q_%d_points", qIdx)))
		if qPoints <= 0 {
			qPoints = 10
		}

		correctOptID, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("q_%d_correct", qIdx)))

		var options []model.Option
		for oIdx := 1; oIdx <= 4; oIdx++ {
			optText := strings.TrimSpace(r.FormValue(fmt.Sprintf("q_%d_opt_%d", qIdx, oIdx)))
			if optText != "" {
				options = append(options, model.Option{
					ID:        oIdx,
					Text:      optText,
					IsCorrect: oIdx == correctOptID,
				})
			}
		}

		if len(options) >= 2 {
			questions = append(questions, model.Question{
				ID:      qIdx,
				Text:    qText,
				Points:  qPoints,
				Options: options,
			})
		}
	}

	if len(questions) == 0 {
		http.Redirect(w, r, "/admin/quizzes/create?error=Please+add+at+least+one+valid+question", http.StatusSeeOther)
		return
	}

	quiz := &model.Quiz{
		ID:              primitive.NewObjectID(),
		Title:           title,
		Description:     description,
		LevelID:         levelID,
		GroupIDs:        groupIDs,
		DurationMinutes: duration,
		StartTime:       time.Now().Add(-10 * time.Minute),
		EndTime:         time.Now().Add(7 * 24 * time.Hour),
		IsActive:        true,
		Questions:       questions,
		CreatedAt:       time.Now().UTC(),
	}

	if err := p.quizService.CreateQuiz(r.Context(), quiz); err != nil {
		http.Error(w, "Failed to create quiz: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (p *AdminPresenter) RenderQuizAnalytics(w http.ResponseWriter, r *http.Request) {
	quizIDHex := strings.TrimPrefix(r.URL.Path, "/admin/quizzes/")
	quizIDHex = strings.TrimSuffix(quizIDHex, "/analytics")

	quizObjID, err := primitive.ObjectIDFromHex(quizIDHex)
	if err != nil {
		http.Error(w, "Invalid Quiz ID", http.StatusBadRequest)
		return
	}

	quiz, err := p.quizService.GetQuizByID(r.Context(), quizObjID)
	if err != nil || quiz == nil {
		http.Error(w, "Quiz not found", http.StatusNotFound)
		return
	}

	results, _ := p.resultService.GetQuizLeaderboard(r.Context(), quizObjID)

	avgScore := 0.0
	maxScore := 0
	if len(results) > 0 {
		total := 0
		for _, r := range results {
			total += r.Score
			if r.Score > maxScore {
				maxScore = r.Score
			}
		}
		avgScore = float64(total) / float64(len(results))
	}

	data := map[string]interface{}{
		"Quiz":         quiz,
		"Results":      results,
		"TotalTakers":  len(results),
		"AverageScore": fmt.Sprintf("%.1f", avgScore),
		"MaxScore":     maxScore,
	}

	_ = p.templates.ExecuteTemplate(w, "admin_analytics.html", data)
}
