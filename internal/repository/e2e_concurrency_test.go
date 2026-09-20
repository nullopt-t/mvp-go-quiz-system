package repository_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"
	"quiz-system/internal/service"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestService_RealMongo_ConcurrentCalculateAndSubmit tests the entire domain pipeline:
// ResultService -> ResultRepo -> MongoDB with 30 concurrent goroutines competing to finalize.
func TestService_RealMongo_ConcurrentCalculateAndSubmit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Ensure unique index as in production
	resultsCol := db.Collection("quiz_results")
	_, err := resultsCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "quiz_id", Value: 1},
			{Key: "student_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		t.Fatalf("failed to create unique index: %v", err)
	}

	quizRepo := repository.NewQuizRepository(db)
	answerRepo := repository.NewAnswerRepository(db)
	resultRepo := repository.NewResultRepository(db)
	resultSvc := service.NewResultService(resultRepo, answerRepo, quizRepo, nil)

	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	// Seed Quiz
	quiz := &model.Quiz{
		ID:              quizID,
		Title:           "Concurrent E2E Exam",
		LevelID:         1,
		GroupIDs:        []string{"A"},
		DurationMinutes: 30,
		StartTime:       time.Now().Add(-10 * time.Minute),
		EndTime:         time.Now().Add(20 * time.Minute),
		IsActive:        true,
		Questions: []model.Question{
			{
				ID:     1,
				Text:   "Question 1",
				Points: 50,
				Options: []model.Option{
					{ID: 10, Text: "Correct Opt 1", IsCorrect: true},
					{ID: 11, Text: "Wrong Opt 1", IsCorrect: false},
				},
			},
			{
				ID:     2,
				Text:   "Question 2",
				Points: 50,
				Options: []model.Option{
					{ID: 20, Text: "Correct Opt 2", IsCorrect: true},
					{ID: 21, Text: "Wrong Opt 2", IsCorrect: false},
				},
			},
		},
	}
	if err := quizRepo.Create(ctx, quiz); err != nil {
		t.Fatalf("failed to create quiz: %v", err)
	}

	// Seed Answers in append-only log: student answered both correctly
	now := time.Now().UTC()
	ans1 := &model.StudentAnswer{
		ID:          primitive.NewObjectID(),
		QuizID:      quizID,
		StudentID:   studentID,
		StudentCode: "E2E-001",
		QuestionID:  1,
		AnswerID:    10,
		SubmittedAt: now,
	}
	ans2 := &model.StudentAnswer{
		ID:          primitive.NewObjectID(),
		QuizID:      quizID,
		StudentID:   studentID,
		StudentCode: "E2E-001",
		QuestionID:  2,
		AnswerID:    20,
		SubmittedAt: now.Add(time.Millisecond),
	}
	if err := answerRepo.AppendAnswer(ctx, ans1); err != nil {
		t.Fatalf("failed to append ans1: %v", err)
	}
	if err := answerRepo.AppendAnswer(ctx, ans2); err != nil {
		t.Fatalf("failed to append ans2: %v", err)
	}

	// Now run 30 concurrent CalculateAndSubmit calls
	const concurrency = 30
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	type callResult struct {
		res *model.QuizResult
		err error
	}
	callResults := make([]callResult, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startBarrier
			res, err := resultSvc.CalculateAndSubmit(ctx, quizID, studentID, "E2E-001", "Concurrent Student", 1, "A")
			callResults[idx] = callResult{res: res, err: err}
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	// Verify all returned canonical result
	var canonicalID primitive.ObjectID
	for i, cr := range callResults {
		if cr.err != nil {
			t.Fatalf("goroutine %d returned error: %v", i, cr.err)
		}
		if cr.res == nil {
			t.Fatalf("goroutine %d returned nil result", i)
		}
		if canonicalID.IsZero() {
			canonicalID = cr.res.ID
		} else if cr.res.ID != canonicalID {
			t.Errorf("goroutine %d got phantom ID %s, expected %s", i, cr.res.ID.Hex(), canonicalID.Hex())
		}
		if cr.res.Score != 100 {
			t.Errorf("goroutine %d got score %d, expected 100", i, cr.res.Score)
		}
	}

	// Verify MongoDB contains exactly 1 document with canonicalID
	docCount, err := resultsCol.CountDocuments(ctx, bson.M{"quiz_id": quizID, "student_id": studentID})
	if err != nil {
		t.Fatalf("CountDocuments failed: %v", err)
	}
	if docCount != 1 {
		t.Errorf("expected 1 document in MongoDB, got %d", docCount)
	}

	dbDoc, err := resultRepo.GetStudentResult(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("GetStudentResult failed: %v", err)
	}
	if dbDoc == nil || dbDoc.ID != canonicalID {
		t.Errorf("MongoDB document ID %v does not match canonical ID %s", dbDoc, canonicalID.Hex())
	}
}
