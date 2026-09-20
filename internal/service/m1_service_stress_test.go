package service_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/service"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Thread-safe mock Result Repository for service stress testing
type threadSafeMockResultRepo struct {
	mu      sync.Mutex
	results map[string]*model.QuizResult
}

func newThreadSafeMockResultRepo() *threadSafeMockResultRepo {
	return &threadSafeMockResultRepo{
		results: make(map[string]*model.QuizResult),
	}
}

func (m *threadSafeMockResultRepo) SaveResult(ctx context.Context, result *model.QuizResult) (*model.QuizResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := result.QuizID.Hex() + ":" + result.StudentID.Hex()
	if existing, ok := m.results[key]; ok {
		*result = *existing
		return existing, nil
	}

	// First insertion becomes canonical
	m.results[key] = result
	return result, nil
}

func (m *threadSafeMockResultRepo) GetStudentResult(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := quizID.Hex() + ":" + studentID.Hex()
	if res, ok := m.results[key]; ok {
		return res, nil
	}
	return nil, nil
}

func (m *threadSafeMockResultRepo) GetStudentResults(ctx context.Context, studentID primitive.ObjectID) ([]model.QuizResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []model.QuizResult
	for _, r := range m.results {
		if r.StudentID == studentID {
			list = append(list, *r)
		}
	}
	return list, nil
}

func (m *threadSafeMockResultRepo) GetQuizResults(ctx context.Context, quizID primitive.ObjectID) ([]model.QuizResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var list []model.QuizResult
	for _, r := range m.results {
		if r.QuizID == quizID {
			list = append(list, *r)
		}
	}
	return list, nil
}

func (m *threadSafeMockResultRepo) GetLevelResults(ctx context.Context, quizID primitive.ObjectID, levelID int, groupID string) ([]model.QuizResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

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

type staticQuizRepo struct {
	quiz *model.Quiz
}

func (s *staticQuizRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Quiz, error) {
	return s.quiz, nil
}
func (s *staticQuizRepo) Create(ctx context.Context, quiz *model.Quiz) error { return nil }
func (s *staticQuizRepo) Update(ctx context.Context, quiz *model.Quiz) error { return nil }
func (s *staticQuizRepo) Delete(ctx context.Context, id primitive.ObjectID) error { return nil }
func (s *staticQuizRepo) GetAll(ctx context.Context) ([]model.Quiz, error)        { return nil, nil }
func (s *staticQuizRepo) GetAvailableForStudent(ctx context.Context, levelID int, groupID string) ([]model.Quiz, error) {
	return nil, nil
}

type staticAnswerRepo struct {
	answers map[int]int
}

func (s *staticAnswerRepo) AppendAnswer(ctx context.Context, answer *model.StudentAnswer) error {
	return nil
}
func (s *staticAnswerRepo) GetStudentAnswers(ctx context.Context, quizID, studentID primitive.ObjectID) ([]model.StudentAnswer, error) {
	return nil, nil
}
func (s *staticAnswerRepo) GetLatestAnswersMap(ctx context.Context, quizID, studentID primitive.ObjectID) (map[int]int, error) {
	return s.answers, nil
}
func (s *staticAnswerRepo) CountStudentAnswers(ctx context.Context, quizID, studentID primitive.ObjectID) (int64, error) {
	return int64(len(s.answers)), nil
}

// TestService_CalculateAndSubmit_HighConcurrencyStress simulates 50 concurrent callers
// calling CalculateAndSubmit at the exact same moment.
func TestService_CalculateAndSubmit_HighConcurrencyStress(t *testing.T) {
	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	quiz := &model.Quiz{
		ID:        quizID,
		Title:     "Stress Finalization Exam",
		LevelID:   1,
		GroupIDs:  []string{"A"},
		StartTime: time.Now().Add(-10 * time.Minute),
		EndTime:   time.Now().Add(10 * time.Minute),
		IsActive:  true,
		Questions: []model.Question{
			{
				ID:     1,
				Points: 50,
				Options: []model.Option{
					{ID: 1, Text: "Opt1", IsCorrect: true},
					{ID: 2, Text: "Opt2", IsCorrect: false},
				},
			},
			{
				ID:     2,
				Points: 50,
				Options: []model.Option{
					{ID: 1, Text: "Opt1", IsCorrect: false},
					{ID: 2, Text: "Opt2", IsCorrect: true},
				},
			},
		},
	}

	resultRepo := newThreadSafeMockResultRepo()
	quizRepo := &staticQuizRepo{quiz: quiz}
	answerRepo := &staticAnswerRepo{
		answers: map[int]int{
			1: 1, // correct
			2: 2, // correct -> 100 points
		},
	}

	resultSvc := service.NewResultService(resultRepo, answerRepo, quizRepo, nil)
	ctx := context.Background()

	const concurrency = 50
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	type outcome struct {
		res *model.QuizResult
		err error
	}
	outcomes := make([]outcome, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startBarrier
			res, err := resultSvc.CalculateAndSubmit(ctx, quizID, studentID, "STU-001", "Stress Student", 1, "A")
			outcomes[idx] = outcome{res: res, err: err}
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	var canonicalID primitive.ObjectID
	for i, out := range outcomes {
		if out.err != nil {
			t.Fatalf("goroutine %d returned unexpected error: %v", i, out.err)
		}
		if out.res == nil {
			t.Fatalf("goroutine %d returned nil result", i)
		}
		if canonicalID.IsZero() {
			canonicalID = out.res.ID
		} else if out.res.ID != canonicalID {
			t.Errorf("goroutine %d returned non-canonical ID: got %s, expected %s",
				i, out.res.ID.Hex(), canonicalID.Hex())
		}
		if out.res.Score != 100 {
			t.Errorf("goroutine %d returned wrong score: %d", i, out.res.Score)
		}
	}
}

// TestService_GetGroupAnalytics_StringGroupID_Comprehensive tests group analytics filtering.
func TestService_GetGroupAnalytics_StringGroupID_Comprehensive(t *testing.T) {
	quizID := primitive.NewObjectID()
	otherQuizID := primitive.NewObjectID()

	resultRepo := newThreadSafeMockResultRepo()
	ctx := context.Background()

	// Seed records across groups with distinct student IDs
	records := []*model.QuizResult{
		{ID: primitive.NewObjectID(), QuizID: quizID, StudentID: primitive.NewObjectID(), LevelID: 1, GroupID: "A", Score: 90},
		{ID: primitive.NewObjectID(), QuizID: quizID, StudentID: primitive.NewObjectID(), LevelID: 1, GroupID: "A", Score: 80},
		{ID: primitive.NewObjectID(), QuizID: quizID, StudentID: primitive.NewObjectID(), LevelID: 1, GroupID: "B", Score: 85},
		{ID: primitive.NewObjectID(), QuizID: quizID, StudentID: primitive.NewObjectID(), LevelID: 2, GroupID: "A", Score: 95},
		{ID: primitive.NewObjectID(), QuizID: otherQuizID, StudentID: primitive.NewObjectID(), LevelID: 1, GroupID: "A", Score: 70},
	}

	for _, r := range records {
		if _, err := resultRepo.SaveResult(ctx, r); err != nil {
			t.Fatalf("failed to save: %v", err)
		}
	}

	svc := service.NewResultService(resultRepo, nil, nil, nil)

	// Case 1: Filter Level 1, Group "A" -> 2 items
	res1A, err := svc.GetGroupAnalytics(ctx, quizID, 1, "A")
	if err != nil || len(res1A) != 2 {
		t.Fatalf("expected 2 results for Level 1 Group A, got %d (err: %v)", len(res1A), err)
	}

	// Case 2: Filter Level 1, Group "  A  " (whitespace) -> 2 items
	res1ATrimmed, err := svc.GetGroupAnalytics(ctx, quizID, 1, "  A  ")
	if err != nil || len(res1ATrimmed) != 2 {
		t.Fatalf("expected 2 results for Level 1 Group '  A  ', got %d (err: %v)", len(res1ATrimmed), err)
	}

	// Case 3: Filter Level 1, all groups (empty groupID) -> 3 items
	res1All, err := svc.GetGroupAnalytics(ctx, quizID, 1, "")
	if err != nil || len(res1All) != 3 {
		t.Fatalf("expected 3 results for Level 1 all groups, got %d (err: %v)", len(res1All), err)
	}

	// Case 4: Filter all levels (levelID = 0), Group "A" -> 3 items (2 from L1, 1 from L2)
	resAllA, err := svc.GetGroupAnalytics(ctx, quizID, 0, "A")
	if err != nil || len(resAllA) != 3 {
		t.Fatalf("expected 3 results for all levels Group A, got %d (err: %v)", len(resAllA), err)
	}

	// Case 5: Non-existent group -> 0 items
	resEmpty, err := svc.GetGroupAnalytics(ctx, quizID, 1, "NON_EXISTENT")
	if err != nil || len(resEmpty) != 0 {
		t.Fatalf("expected 0 results for non-existent group, got %d", len(resEmpty))
	}
}
