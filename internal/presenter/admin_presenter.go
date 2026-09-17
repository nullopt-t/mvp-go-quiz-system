package presenter

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	importService service.ImportService
	templates     *template.Template
}

func NewAdminPresenter(
	quizService service.QuizService,
	resultService service.ResultService,
	studentRepo repository.StudentRepository,
	importService service.ImportService,
	tmpl *template.Template,
) *AdminPresenter {
	return &AdminPresenter{
		quizService:   quizService,
		resultService: resultService,
		studentRepo:   studentRepo,
		importService: importService,
		templates:     tmpl,
	}
}

func (p *AdminPresenter) RenderDashboard(w http.ResponseWriter, r *http.Request) {
	quizzes, _ := p.quizService.GetAllQuizzes(r.Context())
	totalStudents, _ := p.studentRepo.Count(r.Context())

	// Dynamically aggregate all active cohorts from MongoDB
	groupSummaries, err := p.studentRepo.GetCohortDistribution(r.Context())
	if err != nil {
		groupSummaries = nil
	}

	i18nBundle := GetI18n(r)
	successMsg := r.URL.Query().Get("import_success")
	errMsg := r.URL.Query().Get("import_error")

	data := map[string]interface{}{
		"Quizzes":        quizzes,
		"TotalStudents":  totalStudents,
		"GroupSummaries": groupSummaries,
		"ImportSuccess":  successMsg,
		"ImportError":    errMsg,
		"I18n":           i18nBundle,
	}

	_ = p.templates.ExecuteTemplate(w, "admin_dashboard.html", data)
}

func (p *AdminPresenter) HandlePreviewImportStudents(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		http.Redirect(w, r, "/admin?import_error=Failed+to+read+upload+data", http.StatusSeeOther)
		return
	}

	file, _, err := r.FormFile("csv_file")
	if err != nil {
		http.Redirect(w, r, "/admin?import_error=Please+select+a+valid+CSV+file", http.StatusSeeOther)
		return
	}
	defer file.Close()

	// Save temporary copy for reliable large file importing
	tempFile, err := os.CreateTemp("", "roster-*.csv")
	if err != nil {
		http.Redirect(w, r, "/admin?import_error=Failed+to+process+uploaded+file", http.StatusSeeOther)
		return
	}
	tempPath := tempFile.Name()
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		_ = os.Remove(tempPath)
		http.Redirect(w, r, "/admin?import_error=Failed+to+cache+uploaded+roster", http.StatusSeeOther)
		return
	}

	// Rewind to parse
	if _, err := tempFile.Seek(0, 0); err != nil {
		_ = os.Remove(tempPath)
		http.Redirect(w, r, "/admin?import_error=Failed+to+read+cached+roster", http.StatusSeeOther)
		return
	}

	students, err := p.importService.ParseStudentsFromCSV(tempFile)
	if err != nil {
		_ = os.Remove(tempPath)
		http.Redirect(w, r, fmt.Sprintf("/admin?import_error=%s", strings.ReplaceAll(err.Error(), " ", "+")), http.StatusSeeOther)
		return
	}

	displayLimit := 250
	displayStudents := students
	isTruncated := false
	if len(students) > displayLimit {
		displayStudents = students[:displayLimit]
		isTruncated = true
	}

	fileToken := filepath.Base(tempPath)

	i18nBundle := GetI18n(r)
	data := map[string]interface{}{
		"Students":        displayStudents,
		"TotalCount":      len(students),
		"DisplayedCount":  len(displayStudents),
		"IsTruncated":     isTruncated,
		"FileToken":       fileToken,
		"I18n":            i18nBundle,
	}

	_ = p.templates.ExecuteTemplate(w, "admin_import_preview.html", data)
}

func (p *AdminPresenter) HandleConfirmImportStudents(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin?import_error=Invalid+form+data", http.StatusSeeOther)
		return
	}

	fileToken := strings.TrimSpace(r.FormValue("file_token"))
	importMode := strings.TrimSpace(r.FormValue("import_mode")) // "all_file" or "form_data"

	// If large roster import requested directly from cached file
	if fileToken != "" && (importMode == "all_file" || len(r.Form["student_code[]"]) == 0) {
		tempPath := filepath.Join(os.TempDir(), filepath.Clean(fileToken))
		f, err := os.Open(tempPath)
		if err != nil {
			http.Redirect(w, r, "/admin?import_error=Upload+session+expired.+Please+re-upload+the+file", http.StatusSeeOther)
			return
		}
		defer func() {
			_ = f.Close()
			_ = os.Remove(tempPath)
		}()

		count, err := p.importService.ImportStudentsFromCSV(r.Context(), f)
		if err != nil {
			http.Redirect(w, r, fmt.Sprintf("/admin?import_error=%s", strings.ReplaceAll(err.Error(), " ", "+")), http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/admin/students?success=Successfully+imported+%d+students", count), http.StatusSeeOther)
		return
	}

	// Clean up temporary file if present
	if fileToken != "" {
		_ = os.Remove(filepath.Join(os.TempDir(), filepath.Clean(fileToken)))
	}

	codes := r.Form["student_code[]"]
	names := r.Form["name[]"]
	levels := r.Form["level_id[]"]
	groups := r.Form["group_id[]"]

	var students []model.Student
	now := time.Now().UTC()

	for i := 0; i < len(codes); i++ {
		c := strings.ToUpper(strings.TrimSpace(codes[i]))
		n := ""
		if i < len(names) {
			n = strings.TrimSpace(names[i])
		}
		if c == "" || n == "" {
			continue
		}

		lvl := 1
		if i < len(levels) {
			if l, err := strconv.Atoi(levels[i]); err == nil && l >= 1 && l <= 5 {
				lvl = l
			}
		}

		grp := "A"
		if i < len(groups) {
			g := strings.ToUpper(strings.TrimSpace(groups[i]))
			if g != "" {
				grp = g
			}
		}

		students = append(students, model.Student{
			ID:          primitive.NewObjectID(),
			StudentCode: c,
			Name:        n,
			LevelID:     lvl,
			GroupID:     grp,
			IsActive:    true,
			CreatedAt:   now,
		})
	}

	if len(students) == 0 {
		http.Redirect(w, r, "/admin?import_error=No+students+selected+for+import", http.StatusSeeOther)
		return
	}

	count, err := p.importService.BulkImportStudents(r.Context(), students)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin?import_error=%s", strings.ReplaceAll(err.Error(), " ", "+")), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/students?success=Successfully+imported+%d+students", count), http.StatusSeeOther)
}

func (p *AdminPresenter) HandleDownloadSampleCSV(w http.ResponseWriter, r *http.Request) {
	csvData := p.importService.GenerateSampleCSV()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"students_template.csv\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvData)
}

func (p *AdminPresenter) RenderCreateQuiz(w http.ResponseWriter, r *http.Request) {
	i18nBundle := GetI18n(r)
	_ = p.templates.ExecuteTemplate(w, "admin_create_quiz.html", map[string]interface{}{"I18n": i18nBundle})
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

	// Group IDs parsing (e.g. "A, B, C")
	var groupIDs []string
	groupsInput := r.FormValue("group_ids")
	if groupsInput != "" {
		for _, part := range strings.Split(groupsInput, ",") {
			g := strings.ToUpper(strings.TrimSpace(part))
			if g != "" {
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

	i18nBundle := GetI18n(r)

	data := map[string]interface{}{
		"Quiz":         quiz,
		"Results":      results,
		"TotalTakers":  len(results),
		"AverageScore": fmt.Sprintf("%.1f", avgScore),
		"MaxScore":     maxScore,
		"I18n":         i18nBundle,
	}

	_ = p.templates.ExecuteTemplate(w, "admin_analytics.html", data)
}

func (p *AdminPresenter) RenderStudentsList(w http.ResponseWriter, r *http.Request) {
	levelID, _ := strconv.Atoi(r.URL.Query().Get("level_id"))
	groupID := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("group_id")))

	students, err := p.studentRepo.GetAll(r.Context(), levelID, groupID, 200, 0)
	if err != nil {
		students = nil
	}

	cohorts, _ := p.studentRepo.GetCohortDistribution(r.Context())
	totalStudents, _ := p.studentRepo.Count(r.Context())

	i18nBundle := GetI18n(r)
	successMsg := r.URL.Query().Get("success")
	errMsg := r.URL.Query().Get("error")

	data := map[string]interface{}{
		"Students":       students,
		"Cohorts":        cohorts,
		"TotalStudents":  totalStudents,
		"SelectedLevel":  levelID,
		"SelectedGroup":  groupID,
		"Success":        successMsg,
		"Error":          errMsg,
		"I18n":           i18nBundle,
	}

	_ = p.templates.ExecuteTemplate(w, "admin_students.html", data)
}

func (p *AdminPresenter) RenderCreateStudent(w http.ResponseWriter, r *http.Request) {
	i18nBundle := GetI18n(r)
	data := map[string]interface{}{
		"I18n": i18nBundle,
	}
	_ = p.templates.ExecuteTemplate(w, "admin_student_create.html", data)
}

func (p *AdminPresenter) HandleCreateStudent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(r.FormValue("student_code")))
	name := strings.TrimSpace(r.FormValue("name"))
	levelID, _ := strconv.Atoi(r.FormValue("level_id"))
	groupID := strings.ToUpper(strings.TrimSpace(r.FormValue("group_id")))

	if code == "" || name == "" {
		http.Redirect(w, r, "/admin/students/create?error=Student+code+and+name+are+required", http.StatusSeeOther)
		return
	}
	if levelID < 1 || levelID > 5 {
		levelID = 1
	}
	if groupID == "" {
		groupID = "A"
	}

	student := &model.Student{
		StudentCode: code,
		Name:        name,
		LevelID:     levelID,
		GroupID:     groupID,
		IsActive:    true,
	}

	if err := p.studentRepo.CreateOrUpdate(r.Context(), student); err != nil {
		http.Redirect(w, r, "/admin/students/create?error=Failed+to+save+student", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/students?success=Student+saved+successfully", http.StatusSeeOther)
}

func (p *AdminPresenter) RenderStudentProfile(w http.ResponseWriter, r *http.Request) {
	studentIDHex := strings.TrimPrefix(r.URL.Path, "/admin/students/")

	stuObjID, err := primitive.ObjectIDFromHex(studentIDHex)
	if err != nil {
		http.Error(w, "Invalid Student ID", http.StatusBadRequest)
		return
	}

	student, err := p.studentRepo.FindByID(r.Context(), stuObjID)
	if err != nil || student == nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	history, _ := p.resultService.GetStudentHistory(r.Context(), stuObjID)
	i18nBundle := GetI18n(r)

	data := map[string]interface{}{
		"Student": student,
		"History": history,
		"I18n":    i18nBundle,
	}

	_ = p.templates.ExecuteTemplate(w, "admin_student_profile.html", data)
}
