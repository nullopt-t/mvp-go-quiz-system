package repository_test

import (
	"context"
	"testing"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestQuizResult_BSONTagEncoding is a pure unit test verifying domain model BSON types
// without requiring an active MongoDB daemon.
func TestQuizResult_BSONTagEncoding(t *testing.T) {
	result := model.QuizResult{
		ID:             primitive.NewObjectID(),
		QuizID:         primitive.NewObjectID(),
		QuizTitle:      "Test BSON",
		StudentID:      primitive.NewObjectID(),
		StudentCode:    "L1A-001",
		StudentName:    "Alice",
		LevelID:        1,
		GroupID:        "A",
		Score:          95,
		TotalPoints:    100,
		CorrectCount:   9,
		TotalQuestions: 10,
		SubmittedAt:    time.Now().UTC(),
	}

	data, err := bson.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal QuizResult to BSON: %v", err)
	}

	raw := bson.Raw(data)

	// Verify group_id is encoded as BSON String (bsontype.String == 0x02)
	groupIDVal := raw.Lookup("group_id")
	if groupIDVal.Type != bsontype.String {
		t.Fatalf("expected group_id to be encoded as BSON String (0x02), got %v", groupIDVal.Type)
	}
	if groupIDVal.StringValue() != "A" {
		t.Errorf("expected group_id 'A', got '%s'", groupIDVal.StringValue())
	}

	// Verify level_id is encoded as numeric (Int32 or Int64)
	levelIDVal := raw.Lookup("level_id")
	if levelIDVal.Type != bsontype.Int32 && levelIDVal.Type != bsontype.Int64 {
		t.Errorf("expected level_id to be numeric, got %v", levelIDVal.Type)
	}
}

// TestResultRepository_GetLevelResults_StringGroupID tests cohort isolation with string group IDs.
func TestResultRepository_GetLevelResults_StringGroupID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewResultRepository(db)

	quiz1ID := primitive.NewObjectID()
	quiz2ID := primitive.NewObjectID()

	now := time.Now().UTC()

	// Seed 4 results across levels and groups
	results := []model.QuizResult{
		{
			ID:          primitive.NewObjectID(),
			QuizID:      quiz1ID,
			QuizTitle:   "Quiz 1",
			StudentID:   primitive.NewObjectID(),
			StudentCode: "L1A-001",
			StudentName: "Student 1A",
			LevelID:     1,
			GroupID:     "A",
			Score:       70,
			SubmittedAt: now,
		},
		{
			ID:          primitive.NewObjectID(),
			QuizID:      quiz1ID,
			QuizTitle:   "Quiz 1",
			StudentID:   primitive.NewObjectID(),
			StudentCode: "L1B-001",
			StudentName: "Student 1B",
			LevelID:     1,
			GroupID:     "B",
			Score:       85,
			SubmittedAt: now.Add(time.Second),
		},
		{
			ID:          primitive.NewObjectID(),
			QuizID:      quiz1ID,
			QuizTitle:   "Quiz 1",
			StudentID:   primitive.NewObjectID(),
			StudentCode: "L2A-001",
			StudentName: "Student 2A",
			LevelID:     2,
			GroupID:     "A",
			Score:       95,
			SubmittedAt: now.Add(2 * time.Second),
		},
		{
			ID:          primitive.NewObjectID(),
			QuizID:      quiz2ID,
			QuizTitle:   "Quiz 2",
			StudentID:   primitive.NewObjectID(),
			StudentCode: "L1A-002",
			StudentName: "Student 1A Quiz2",
			LevelID:     1,
			GroupID:     "A",
			Score:       100,
			SubmittedAt: now.Add(3 * time.Second),
		},
	}

	for _, r := range results {
		res := r
		if _, err := repo.SaveResult(ctx, &res); err != nil {
			t.Fatalf("failed to seed result: %v", err)
		}
	}

	// 1. Query Quiz 1, Level 1, Group "A" -> exactly 1 result (Student 1A, score 70)
	q1L1A, err := repo.GetLevelResults(ctx, quiz1ID, 1, "A")
	if err != nil {
		t.Fatalf("GetLevelResults failed: %v", err)
	}
	if len(q1L1A) != 1 {
		t.Fatalf("expected 1 result for Quiz 1, Level 1, Group A, got %d", len(q1L1A))
	}
	if q1L1A[0].StudentCode != "L1A-001" || q1L1A[0].Score != 70 {
		t.Errorf("unexpected result: %+v", q1L1A[0])
	}

	// 2. Query Quiz 1, Level 1, Group "B" -> exactly 1 result (Student 1B, score 85)
	q1L1B, err := repo.GetLevelResults(ctx, quiz1ID, 1, "B")
	if err != nil {
		t.Fatalf("GetLevelResults failed: %v", err)
	}
	if len(q1L1B) != 1 {
		t.Fatalf("expected 1 result for Quiz 1, Level 1, Group B, got %d", len(q1L1B))
	}
	if q1L1B[0].StudentCode != "L1B-001" || q1L1B[0].Score != 85 {
		t.Errorf("unexpected result: %+v", q1L1B[0])
	}

	// 3. Query Quiz 1, Level 1, empty group "" -> 2 results, sorted by score DESC (85 then 70)
	q1L1All, err := repo.GetLevelResults(ctx, quiz1ID, 1, "")
	if err != nil {
		t.Fatalf("GetLevelResults failed: %v", err)
	}
	if len(q1L1All) != 2 {
		t.Fatalf("expected 2 results for Quiz 1, Level 1 all groups, got %d", len(q1L1All))
	}
	if q1L1All[0].Score != 85 || q1L1All[1].Score != 70 {
		t.Errorf("expected sorted scores [85, 70], got [%d, %d]", q1L1All[0].Score, q1L1All[1].Score)
	}

	// 4. Query Quiz 1, Level 0, Group "A" -> 2 results across levels (Level 2 Score 95, Level 1 Score 70)
	q1AllLvlA, err := repo.GetLevelResults(ctx, quiz1ID, 0, "A")
	if err != nil {
		t.Fatalf("GetLevelResults failed: %v", err)
	}
	if len(q1AllLvlA) != 2 {
		t.Fatalf("expected 2 results for Quiz 1, all levels, Group A, got %d", len(q1AllLvlA))
	}
	if q1AllLvlA[0].Score != 95 || q1AllLvlA[1].Score != 70 {
		t.Errorf("expected sorted scores [95, 70], got [%d, %d]", q1AllLvlA[0].Score, q1AllLvlA[1].Score)
	}

	// 5. Query Quiz 1, whitespace group " A " -> trimmed to "A"
	q1Trimmed, err := repo.GetLevelResults(ctx, quiz1ID, 1, " A ")
	if err != nil {
		t.Fatalf("GetLevelResults failed: %v", err)
	}
	if len(q1Trimmed) != 1 || q1Trimmed[0].StudentCode != "L1A-001" {
		t.Errorf("expected 1 result matching trimmed group 'A', got %+v", q1Trimmed)
	}
}

// TestResultRepository_SaveResult_ConcurrentCanonicalReload tests that when an upsert
// encounters an existing document, it reloads and returns the canonical persisted state.
func TestResultRepository_SaveResult_ConcurrentCanonicalReload(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewResultRepository(db)

	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	canonicalID := primitive.NewObjectID()
	initialResult := &model.QuizResult{
		ID:           canonicalID,
		QuizID:       quizID,
		QuizTitle:    "Concurrency Exam",
		StudentID:    studentID,
		StudentCode:  "L1A-CONCURRENT",
		StudentName:  "Test Student",
		LevelID:      1,
		GroupID:      "A",
		Score:        90,
		TotalPoints:  100,
		CorrectCount: 9,
		SubmittedAt:  time.Now().UTC(),
	}

	// 1. Initial save succeeds
	saved1, err := repo.SaveResult(ctx, initialResult)
	if err != nil {
		t.Fatalf("initial SaveResult failed: %v", err)
	}
	if saved1.ID != canonicalID {
		t.Errorf("expected initial saved ID %s, got %s", canonicalID.Hex(), saved1.ID.Hex())
	}

	// 2. Simulate concurrent submission with a conflicting in-memory struct
	conflictingID := primitive.NewObjectID()
	conflictingResult := &model.QuizResult{
		ID:           conflictingID,
		QuizID:       quizID,
		QuizTitle:    "Concurrency Exam",
		StudentID:    studentID,
		StudentCode:  "L1A-CONCURRENT",
		StudentName:  "Test Student",
		LevelID:      1,
		GroupID:      "A",
		Score:        40, // Divergent score
		TotalPoints:  100,
		CorrectCount: 4,
		SubmittedAt:  time.Now().UTC().Add(time.Second),
	}

	saved2, err := repo.SaveResult(ctx, conflictingResult)
	if err != nil {
		t.Fatalf("second SaveResult failed: %v", err)
	}

	// 3. Assert canonical persisted document is returned (NOT the phantom candidate)
	if saved2.ID != canonicalID {
		t.Errorf("SaveResult returned phantom ID %s instead of canonical ID %s", saved2.ID.Hex(), canonicalID.Hex())
	}
	if saved2.Score != 90 {
		t.Errorf("SaveResult returned wrong score %d instead of canonical score 90", saved2.Score)
	}
	if conflictingResult.ID != canonicalID {
		t.Errorf("SaveResult did not reload canonical ID into passed pointer: got %s, expected %s",
			conflictingResult.ID.Hex(), canonicalID.Hex())
	}

	// 4. Verify MongoDB collection still contains the canonical record
	fetched, err := repo.GetStudentResult(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("GetStudentResult failed: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected non-nil fetched result")
	}
	if fetched.ID != canonicalID || fetched.Score != 90 {
		t.Errorf("MongoDB document corrupted: ID=%s, Score=%d (expected ID=%s, Score=90)",
			fetched.ID.Hex(), fetched.Score, canonicalID.Hex())
	}
}
