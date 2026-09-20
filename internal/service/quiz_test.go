package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"quiz-system/internal/config"
	"quiz-system/internal/model"
	"quiz-system/internal/repository"
	"quiz-system/internal/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockStudentRepo
type mockStudentRepo struct {
	students map[string]*model.Student
}

func (m *mockStudentRepo) FindByCode(ctx context.Context, code string) (*model.Student, error) {
	return m.students[code], nil
}
func (m *mockStudentRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Student, error) {
	for _, s := range m.students {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}
func (m *mockStudentRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(m.students)), nil
}
func (m *mockStudentRepo) CountByLevelAndGroup(ctx context.Context, levelID int, groupID string) (int64, error) {
	count := int64(0)
	for _, s := range m.students {
		if (levelID == 0 || s.LevelID == levelID) && (groupID == "" || s.GroupID == groupID) {
			count++
		}
	}
	return count, nil
}
func (m *mockStudentRepo) BulkInsert(ctx context.Context, students []model.Student) error {
	for i := range students {
		m.students[students[i].StudentCode] = &students[i]
	}
	return nil
}
func (m *mockStudentRepo) BulkUpsert(ctx context.Context, students []model.Student) (int, error) {
	for i := range students {
		m.students[students[i].StudentCode] = &students[i]
	}
	return len(students), nil
}
func (m *mockStudentRepo) CreateOrUpdate(ctx context.Context, student *model.Student) error {
	m.students[student.StudentCode] = student
	return nil
}
func (m *mockStudentRepo) GetCohortDistribution(ctx context.Context) ([]repository.CohortCount, error) {
	return nil, nil
}
func (m *mockStudentRepo) GetAll(ctx context.Context, levelID int, groupID string, limit, offset int64) ([]model.Student, error) {
	var list []model.Student
	for _, s := range m.students {
		if (levelID == 0 || s.LevelID == levelID) && (groupID == "" || s.GroupID == groupID) {
			list = append(list, *s)
		}
	}
	return list, nil
}
func (m *mockStudentRepo) GetAllSorted(ctx context.Context, levelID int, groupID string, sortBy string, sortOrder int, limit, offset int64) ([]model.Student, error) {
	return m.GetAll(ctx, levelID, groupID, limit, offset)
}
func (m *mockStudentRepo) GetByCursor(ctx context.Context, levelID int, groupID string, sortBy string, sortOrder int, cursorVal string, cursorID primitive.ObjectID, direction string, limit int64) ([]model.Student, bool, error) {
	list, err := m.GetAll(ctx, levelID, groupID, limit, 0)
	return list, false, err
}
func (m *mockStudentRepo) SetActive(ctx context.Context, id primitive.ObjectID, isActive bool) error {
	for _, s := range m.students {
		if s.ID == id {
			s.IsActive = isActive
			return nil
		}
	}
	return nil
}

// MockQuizRepo
type mockQuizRepo struct {
	quizzes map[primitive.ObjectID]*model.Quiz
}

func (m *mockQuizRepo) Create(ctx context.Context, quiz *model.Quiz) error {
	if quiz.ID.IsZero() {
		quiz.ID = primitive.NewObjectID()
	}
	m.quizzes[quiz.ID] = quiz
	return nil
}
func (m *mockQuizRepo) Update(ctx context.Context, quiz *model.Quiz) error {
	m.quizzes[quiz.ID] = quiz
	return nil
}
func (m *mockQuizRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Quiz, error) {
	return m.quizzes[id], nil
}
func (m *mockQuizRepo) GetAll(ctx context.Context) ([]model.Quiz, error) {
	var list []model.Quiz
	for _, q := range m.quizzes {
		list = append(list, *q)
	}
	return list, nil
}
func (m *mockQuizRepo) GetAvailableForStudent(ctx context.Context, levelID int, groupID string) ([]model.Quiz, error) {
	return m.GetAll(ctx)
}
func (m *mockQuizRepo) Delete(ctx context.Context, id primitive.ObjectID) error {
	delete(m.quizzes, id)
	return nil
}

// MockAnswerRepo (Simulating pure append-only)
type mockAnswerRepo struct {
	answers []model.StudentAnswer
}

func (m *mockAnswerRepo) AppendAnswer(ctx context.Context, answer *model.StudentAnswer) error {
	m.answers = append(m.answers, *answer)
	return nil
}
func (m *mockAnswerRepo) GetStudentAnswers(ctx context.Context, quizID, studentID primitive.ObjectID) ([]model.StudentAnswer, error) {
	var list []model.StudentAnswer
	for _, a := range m.answers {
		if a.QuizID == quizID && a.StudentID == studentID {
			list = append(list, a)
		}
	}
	return list, nil
}
func (m *mockAnswerRepo) GetLatestAnswersMap(ctx context.Context, quizID, studentID primitive.ObjectID) (map[int]int, error) {
	ans, _ := m.GetStudentAnswers(ctx, quizID, studentID)
	result := make(map[int]int)
	for _, a := range ans {
		result[a.QuestionID] = a.AnswerID
	}
	return result, nil
}
func (m *mockAnswerRepo) CountStudentAnswers(ctx context.Context, quizID, studentID primitive.ObjectID) (int64, error) {
	ans, _ := m.GetStudentAnswers(ctx, quizID, studentID)
	return int64(len(ans)), nil
}

// MockSessionRepo
type mockSessionRepo struct {
	sessions map[string]*model.QuizSessionRecord
}

func (m *mockSessionRepo) StartSessionIfAbsent(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizSessionRecord, error) {
	key := quizID.Hex() + ":" + studentID.Hex()
	if s, exists := m.sessions[key]; exists {
		return s, nil
	}
	s := &model.QuizSessionRecord{
		ID:        primitive.NewObjectID(),
		QuizID:    quizID,
		StudentID: studentID,
		StartedAt: time.Now().UTC(),
	}
	m.sessions[key] = s
	return s, nil
}
func (m *mockSessionRepo) GetSession(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizSessionRecord, error) {
	return m.sessions[quizID.Hex()+":"+studentID.Hex()], nil
}

// MockResultRepo
type mockResultRepo struct {
	results map[string]*model.QuizResult
}

func (m *mockResultRepo) SaveResult(ctx context.Context, result *model.QuizResult) (*model.QuizResult, error) {
	key := result.QuizID.Hex() + ":" + result.StudentID.Hex()
	if existing, ok := m.results[key]; ok {
		*result = *existing
		return existing, nil
	}
	m.results[key] = result
	return result, nil
}
func (m *mockResultRepo) GetStudentResult(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizResult, error) {
	return m.results[quizID.Hex()+":"+studentID.Hex()], nil
}
func (m *mockResultRepo) GetStudentResults(ctx context.Context, studentID primitive.ObjectID) ([]model.QuizResult, error) {
	var list []model.QuizResult
	for _, r := range m.results {
		if r.StudentID == studentID {
			list = append(list, *r)
		}
	}
	return list, nil
}
func (m *mockResultRepo) GetQuizResults(ctx context.Context, quizID primitive.ObjectID) ([]model.QuizResult, error) {
	var list []model.QuizResult
	for _, r := range m.results {
		if r.QuizID == quizID {
			list = append(list, *r)
		}
	}
	return list, nil
}
func (m *mockResultRepo) GetLevelResults(ctx context.Context, quizID primitive.ObjectID, levelID int, groupID string) ([]model.QuizResult, error) {
	var list []model.QuizResult
	trimmedGroup := strings.TrimSpace(groupID)
	for _, r := range m.results {
		if r.QuizID == quizID &&
			(levelID <= 0 || r.LevelID == levelID) &&
			(trimmedGroup == "" || r.GroupID == trimmedGroup) {
			list = append(list, *r)
		}
	}
	return list, nil
}

func TestAuthAndAppendOnlyFlow(t *testing.T) {
	cfg := &config.Config{
		JWTSecret: []byte("test-secret-key-1234567890"),
		JWTExpiry: 15 * time.Minute,
		AdminPIN:  "admin123",
	}

	studentID := primitive.NewObjectID()
	studentRepo := &mockStudentRepo{
		students: map[string]*model.Student{
			"L1A-001": {
				ID:          studentID,
				StudentCode: "L1A-001",
				Name:        "Alice Student",
				LevelID:     1,
				GroupID:     "A",
				IsActive:    true,
			},
		},
	}

	quizID := primitive.NewObjectID()
	quizRepo := &mockQuizRepo{
		quizzes: map[primitive.ObjectID]*model.Quiz{
			quizID: {
				ID:              quizID,
				Title:           "Algorithm Midterm",
				LevelID:         1,
				DurationMinutes: 10,
				StartTime:       time.Now().Add(-1 * time.Minute),
				IsActive:        true,
				Questions: []model.Question{
					{
						ID:     1,
						Text:   "Binary Search complexity?",
						Points: 10,
						Options: []model.Option{
							{ID: 1, Text: "O(n)", IsCorrect: false},
							{ID: 2, Text: "O(log n)", IsCorrect: true},
						},
					},
					{
						ID:     2,
						Text:   "Stack ordering?",
						Points: 10,
						Options: []model.Option{
							{ID: 1, Text: "LIFO", IsCorrect: true},
							{ID: 2, Text: "FIFO", IsCorrect: false},
						},
					},
				},
			},
		},
	}

	answerRepo := &mockAnswerRepo{}
	sessionRepo := &mockSessionRepo{sessions: make(map[string]*model.QuizSessionRecord)}
	resultRepo := &mockResultRepo{results: make(map[string]*model.QuizResult)}

	authSvc := service.NewAuthService(studentRepo, cfg)
	quizSvc := service.NewQuizService(quizRepo, sessionRepo, answerRepo, resultRepo)
	answerSvc := service.NewAnswerService(answerRepo, quizRepo, sessionRepo, resultRepo)
	resultSvc := service.NewResultService(resultRepo, answerRepo, quizRepo, studentRepo)

	ctx := context.Background()

	// 1. Test Auth
	token, student, err := authSvc.LoginStudent(ctx, "L1A-001")
	if err != nil || student == nil || token == "" {
		t.Fatalf("LoginStudent failed: %v", err)
	}

	claims, err := authSvc.ValidateToken(token)
	if err != nil || claims.StudentCode != "L1A-001" {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	// 2. Start Quiz
	quizView, err := quizSvc.StartOrResumeQuiz(ctx, quizID, studentID)
	if err != nil || quizView.RemainingSeconds <= 0 {
		t.Fatalf("StartOrResumeQuiz failed: %v", err)
	}

	// Check that correct options are sanitized
	for _, q := range quizView.Quiz.Questions {
		for _, opt := range q.Options {
			if opt.IsCorrect {
				t.Fatalf("Quiz view must sanitize IsCorrect boolean for student security")
			}
		}
	}

	// 3. Append-Only Answering (Student answers Q1, then changes answer to Q1, then answers Q2)
	err = answerSvc.RecordAnswer(ctx, quizID, studentID, "L1A-001", 1, 1) // First chose option 1 (incorrect)
	if err != nil {
		t.Fatalf("RecordAnswer 1 failed: %v", err)
	}
	err = answerSvc.RecordAnswer(ctx, quizID, studentID, "L1A-001", 1, 2) // Changed to option 2 (correct)
	if err != nil {
		t.Fatalf("RecordAnswer 2 failed: %v", err)
	}
	err = answerSvc.RecordAnswer(ctx, quizID, studentID, "L1A-001", 2, 1) // Chose option 1 for Q2 (correct)
	if err != nil {
		t.Fatalf("RecordAnswer 3 failed: %v", err)
	}

	// Verify append-only storage count: must be 3 total records (never in-place overwritten)
	if len(answerRepo.answers) != 3 {
		t.Fatalf("Expected 3 append-only documents in log, got %d", len(answerRepo.answers))
	}

	// 4. Calculate Final Result
	result, err := resultSvc.CalculateAndSubmit(ctx, quizID, studentID, "L1A-001", "Alice Student", 1, "A")
	if err != nil {
		t.Fatalf("CalculateAndSubmit failed: %v", err)
	}

	// Alice got both Q1 (changed to 2) and Q2 (chose 1) correct = 20/20 points
	if result.Score != 20 || result.CorrectCount != 2 {
		t.Fatalf("Expected 20 points and 2 correct answers, got %d points, %d correct", result.Score, result.CorrectCount)
	}
}

func TestCSVImportService(t *testing.T) {
	studentRepo := &mockStudentRepo{
		students: make(map[string]*model.Student),
	}
	importSvc := service.NewImportService(studentRepo)

	ctx := context.Background()
	sampleCSV := importSvc.GenerateSampleCSV()

	// 1. Test Parse preview
	parsedStudents, err := importSvc.ParseStudentsFromCSV(strings.NewReader(string(sampleCSV)))
	if err != nil {
		t.Fatalf("ParseStudentsFromCSV failed: %v", err)
	}
	if len(parsedStudents) != 6 {
		t.Fatalf("Expected 6 parsed students for preview, got %d", len(parsedStudents))
	}

	// 2. Test Bulk Import from preview list
	count, err := importSvc.BulkImportStudents(ctx, parsedStudents)
	if err != nil {
		t.Fatalf("BulkImportStudents failed: %v", err)
	}

	if count != 6 {
		t.Fatalf("Expected 6 imported students, got %d", count)
	}

	stu, err := studentRepo.FindByCode(ctx, "L1A-001")
	if err != nil || stu == nil {
		t.Fatalf("Expected student L1A-001 to exist")
	}
	if stu.GroupID != "A" || stu.LevelID != 1 {
		t.Fatalf("Expected Level 1, Group A, got Level %d, Group %s", stu.LevelID, stu.GroupID)
	}
}

func TestCalculateAndSubmit_ConcurrentFinalizationPhantomPrevention(t *testing.T) {
	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	quizRepo := &mockQuizRepo{
		quizzes: map[primitive.ObjectID]*model.Quiz{
			quizID: {
				ID:        quizID,
				Title:     "Test Quiz",
				LevelID:   1,
				StartTime: time.Now().Add(-5 * time.Minute),
				IsActive:  true,
				Questions: []model.Question{
					{
						ID:     1,
						Points: 10,
						Options: []model.Option{
							{ID: 1, IsCorrect: true},
						},
					},
				},
			},
		},
	}
	studentRepo := &mockStudentRepo{students: make(map[string]*model.Student)}
	answerRepo := &mockAnswerRepo{}
	resultRepo := &mockResultRepo{results: make(map[string]*model.QuizResult)}

	resultSvc := service.NewResultService(resultRepo, answerRepo, quizRepo, studentRepo)
	ctx := context.Background()

	// Pre-seed an existing canonical result in the repo (simulating a concurrent submission that won the race)
	canonicalID := primitive.NewObjectID()
	canonicalResult := &model.QuizResult{
		ID:           canonicalID,
		QuizID:       quizID,
		StudentID:    studentID,
		StudentCode:  "L1A-001",
		StudentName:  "Alice",
		LevelID:      1,
		GroupID:      "A",
		Score:        100,
		TotalPoints:  100,
		CorrectCount: 1,
		SubmittedAt:  time.Now().UTC().Add(-10 * time.Second),
	}
	key := quizID.Hex() + ":" + studentID.Hex()
	resultRepo.results[key] = canonicalResult

	// Now call SaveResult with a candidate result (different in-memory ID and score)
	candidateID := primitive.NewObjectID()
	candidateResult := &model.QuizResult{
		ID:           candidateID,
		QuizID:       quizID,
		StudentID:    studentID,
		StudentCode:  "L1A-001",
		StudentName:  "Alice",
		LevelID:      1,
		GroupID:      "A",
		Score:        50,
		TotalPoints:  100,
		CorrectCount: 0,
		SubmittedAt:  time.Now().UTC(),
	}

	saved, err := resultRepo.SaveResult(ctx, candidateResult)
	if err != nil {
		t.Fatalf("SaveResult failed: %v", err)
	}

	// Must return the canonical persisted result, NOT the candidate phantom pointer
	if saved.ID != canonicalID {
		t.Errorf("Expected saved result to have canonical ID %s, got phantom ID %s", canonicalID.Hex(), saved.ID.Hex())
	}
	if saved.Score != 100 {
		t.Errorf("Expected canonical score 100, got %d", saved.Score)
	}
	// candidateResult itself should have been updated in-place to canonical state
	if candidateResult.ID != canonicalID {
		t.Errorf("Expected candidateResult pointer to be updated to canonical ID %s, got %s", canonicalID.Hex(), candidateResult.ID.Hex())
	}

	// Also verify CalculateAndSubmit returns the canonical result
	calcResult, err := resultSvc.CalculateAndSubmit(ctx, quizID, studentID, "L1A-001", "Alice", 1, "A")
	if err != nil {
		t.Fatalf("CalculateAndSubmit failed: %v", err)
	}
	if calcResult.ID != canonicalID {
		t.Errorf("Expected CalculateAndSubmit to return canonical ID %s, got %s", canonicalID.Hex(), calcResult.ID.Hex())
	}
}

func TestResultService_GetGroupAnalytics_StringGroupID(t *testing.T) {
	quizID := primitive.NewObjectID()
	otherQuizID := primitive.NewObjectID()

	resultRepo := &mockResultRepo{
		results: map[string]*model.QuizResult{
			"1": {
				ID:      primitive.NewObjectID(),
				QuizID:  quizID,
				LevelID: 1,
				GroupID: "A",
				Score:   70,
			},
			"2": {
				ID:      primitive.NewObjectID(),
				QuizID:  quizID,
				LevelID: 1,
				GroupID: "B",
				Score:   85,
			},
			"3": {
				ID:      primitive.NewObjectID(),
				QuizID:  quizID,
				LevelID: 2,
				GroupID: "A",
				Score:   90,
			},
			"4": {
				ID:      primitive.NewObjectID(),
				QuizID:  otherQuizID,
				LevelID: 1,
				GroupID: "A",
				Score:   100,
			},
		},
	}
	resultSvc := service.NewResultService(resultRepo, nil, nil, nil)
	ctx := context.Background()

	// Query Level 1, Group "A"
	res1A, err := resultSvc.GetGroupAnalytics(ctx, quizID, 1, "A")
	if err != nil {
		t.Fatalf("GetGroupAnalytics failed: %v", err)
	}
	if len(res1A) != 1 || res1A[0].GroupID != "A" || res1A[0].LevelID != 1 {
		t.Fatalf("Expected 1 result for Level 1 Group A, got %+v", res1A)
	}

	// Query Level 1, Group "B"
	res1B, err := resultSvc.GetGroupAnalytics(ctx, quizID, 1, "B")
	if err != nil {
		t.Fatalf("GetGroupAnalytics failed: %v", err)
	}
	if len(res1B) != 1 || res1B[0].GroupID != "B" {
		t.Fatalf("Expected 1 result for Level 1 Group B, got %+v", res1B)
	}

	// Query with whitespace " A "
	resTrimmed, err := resultSvc.GetGroupAnalytics(ctx, quizID, 1, " A ")
	if err != nil {
		t.Fatalf("GetGroupAnalytics with trimmed group failed: %v", err)
	}
	if len(resTrimmed) != 1 || resTrimmed[0].GroupID != "A" {
		t.Fatalf("Expected 1 result for trimmed Group ' A ', got %+v", resTrimmed)
	}

	// Query Level 1, empty group (all groups in Level 1)
	resAllLevel1, err := resultSvc.GetGroupAnalytics(ctx, quizID, 1, "")
	if err != nil {
		t.Fatalf("GetGroupAnalytics all groups failed: %v", err)
	}
	if len(resAllLevel1) != 2 {
		t.Fatalf("Expected 2 results for Level 1 all groups, got %d", len(resAllLevel1))
	}
}

