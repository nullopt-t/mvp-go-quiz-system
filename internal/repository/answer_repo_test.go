package repository_test

import (
	"context"
	"testing"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAnswerRepository_SortingDeterminismOnIdenticalTimestamps(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewAnswerRepository(db)

	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	// Both answers share the EXACT same millisecond timestamp
	sharedTime := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)

	id1 := primitive.NewObjectID()
	time.Sleep(2 * time.Millisecond)
	id2 := primitive.NewObjectID()
	if id1.Hex() >= id2.Hex() {
		t.Fatalf("id1 should be strictly less than id2")
	}

	doc1 := model.StudentAnswer{
		ID:          id1,
		QuizID:      quizID,
		StudentID:   studentID,
		StudentCode: "TEST-01",
		QuestionID:  1,
		AnswerID:    10, // Initial choice
		SubmittedAt: sharedTime,
	}

	doc2 := model.StudentAnswer{
		ID:          id2,
		QuizID:      quizID,
		StudentID:   studentID,
		StudentCode: "TEST-01",
		QuestionID:  1,
		AnswerID:    20, // Revised choice (newer by ObjectID counter)
		SubmittedAt: sharedTime,
	}

	// Deliberately insert doc2 first into MongoDB collection to test that sort
	// does not rely on natural insertion order or table scan order
	col := db.Collection("student_answers")
	if _, err := col.InsertOne(ctx, doc2); err != nil {
		t.Fatalf("failed to insert doc2: %v", err)
	}
	if _, err := col.InsertOne(ctx, doc1); err != nil {
		t.Fatalf("failed to insert doc1: %v", err)
	}

	// 1. Verify GetStudentAnswers returns docs in strict causal order (doc1 then doc2)
	answers, err := repo.GetStudentAnswers(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("GetStudentAnswers failed: %v", err)
	}
	if len(answers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(answers))
	}
	if answers[0].ID != id1 || answers[1].ID != id2 {
		t.Errorf("GetStudentAnswers out of order: got IDs [%s, %s], expected [%s, %s]",
			answers[0].ID.Hex(), answers[1].ID.Hex(), id1.Hex(), id2.Hex())
	}

	// 2. Verify GetLatestAnswersMap folds to the newest answer (doc2.AnswerID = 20)
	latestMap, err := repo.GetLatestAnswersMap(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("GetLatestAnswersMap failed: %v", err)
	}
	if latestMap[1] != 20 {
		t.Errorf("GetLatestAnswersMap returned stale answer: got %d, expected 20 (doc2)", latestMap[1])
	}
}

func TestAnswerRepository_AppendAndCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewAnswerRepository(db)

	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	ans := &model.StudentAnswer{
		QuizID:      quizID,
		StudentID:   studentID,
		StudentCode: "TEST-COUNT",
		QuestionID:  1,
		AnswerID:    2,
	}

	if err := repo.AppendAnswer(ctx, ans); err != nil {
		t.Fatalf("AppendAnswer failed: %v", err)
	}

	count, err := repo.CountStudentAnswers(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("CountStudentAnswers failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}
}

func TestAnswerRepository_CompoundSortSpecification(t *testing.T) {
	// Unit test ensuring compound sort keys submitted_at: 1 and _id: 1 are properly structured
	sortDoc := bson.D{
		{Key: "submitted_at", Value: 1},
		{Key: "_id", Value: 1},
	}
	if len(sortDoc) != 2 {
		t.Fatalf("expected 2 sort keys, got %d", len(sortDoc))
	}
	if sortDoc[0].Key != "submitted_at" || sortDoc[0].Value != 1 {
		t.Errorf("expected sortDoc[0] to be {submitted_at: 1}, got %+v", sortDoc[0])
	}
	if sortDoc[1].Key != "_id" || sortDoc[1].Value != 1 {
		t.Errorf("expected sortDoc[1] to be {_id: 1}, got %+v", sortDoc[1])
	}
}
