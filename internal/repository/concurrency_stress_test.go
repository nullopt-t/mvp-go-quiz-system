package repository_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestSaveResult_HighConcurrencyStress tests SaveResult under high concurrency
// with the production unique index present on (quiz_id, student_id).
func TestSaveResult_HighConcurrencyStress(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Ensure production unique index on quiz_results (quiz_id, student_id)
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

	repo := repository.NewResultRepository(db)

	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	const concurrency = 50
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	type callOutcome struct {
		result *model.QuizResult
		input  *model.QuizResult
		err    error
	}

	outcomes := make([]callOutcome, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			input := &model.QuizResult{
				ID:             primitive.NewObjectID(),
				QuizID:         quizID,
				QuizTitle:      "Concurrency Stress",
				StudentID:      studentID,
				StudentCode:    "STRESS-001",
				StudentName:    "Concurrent Alice",
				LevelID:        1,
				GroupID:        "A",
				Score:          idx * 10,
				TotalPoints:    100,
				CorrectCount:   idx,
				TotalQuestions: 10,
				SubmittedAt:    time.Now().UTC(),
			}

			<-startBarrier

			res, err := repo.SaveResult(ctx, input)
			outcomes[idx] = callOutcome{
				result: res,
				input:  input,
				err:    err,
			}
		}(i)
	}

	// Release all goroutines simultaneously
	close(startBarrier)
	wg.Wait()

	// Check outcomes
	var canonicalID primitive.ObjectID
	var canonicalScore int
	var errCount int

	for i, out := range outcomes {
		if out.err != nil {
			errCount++
			t.Logf("Goroutine %d returned error: %v", i, out.err)
			continue
		}
		if out.result == nil {
			t.Errorf("Goroutine %d returned nil result with nil err", i)
			continue
		}
		if canonicalID.IsZero() {
			canonicalID = out.result.ID
			canonicalScore = out.result.Score
			t.Logf("Established canonical result: ID=%s, Score=%d (from goroutine %d)", canonicalID.Hex(), canonicalScore, i)
		} else {
			if out.result.ID != canonicalID {
				t.Errorf("PHANTOM RESULT DETECTED: Goroutine %d got ID=%s, expected canonical ID=%s",
					i, out.result.ID.Hex(), canonicalID.Hex())
			}
			if out.result.Score != canonicalScore {
				t.Errorf("DIVERGENT SCORE DETECTED: Goroutine %d got Score=%d, expected canonical Score=%d",
					i, out.result.Score, canonicalScore)
			}
			if out.input.ID != canonicalID {
				t.Errorf("INPUT STRUCT NOT MUTATED: Goroutine %d input.ID=%s, expected %s",
					i, out.input.ID.Hex(), canonicalID.Hex())
			}
		}
	}

	if errCount > 0 {
		t.Errorf("SaveResult failed under concurrency: %d of %d calls failed with error", errCount, concurrency)
	}

	// Verify DB state: exactly 1 document
	count, err := resultsCol.CountDocuments(ctx, bson.M{"quiz_id": quizID, "student_id": studentID})
	if err != nil {
		t.Fatalf("CountDocuments failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected exactly 1 document in DB, found %d", count)
	}
}
