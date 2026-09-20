package repository_test

import (
	"context"
	"fmt"
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

// TestMilestone1_SaveResult_NoPhantomResultsStress runs concurrent SaveResult
// requests under extreme race conditions to ensure that no caller receives a phantom in-memory pointer.
func TestMilestone1_SaveResult_NoPhantomResultsStress(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

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

	const numStudents = 5
	const callersPerStudent = 20

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	type resultRecord struct {
		studentIdx int
		returnedID primitive.ObjectID
		inputID    primitive.ObjectID
		score      int
		err        error
	}

	totalCalls := numStudents * callersPerStudent
	outcomes := make([]resultRecord, totalCalls)

	studentIDs := make([]primitive.ObjectID, numStudents)
	quizID := primitive.NewObjectID()

	for s := 0; s < numStudents; s++ {
		studentIDs[s] = primitive.NewObjectID()
	}

	callIdx := 0
	for s := 0; s < numStudents; s++ {
		for c := 0; c < callersPerStudent; c++ {
			wg.Add(1)
			go func(idx, sIdx, cIdx int) {
				defer wg.Done()
				input := &model.QuizResult{
					ID:             primitive.NewObjectID(),
					QuizID:         quizID,
					StudentID:      studentIDs[sIdx],
					StudentCode:    fmt.Sprintf("STUDENT-%d", sIdx),
					StudentName:    fmt.Sprintf("Student %d", sIdx),
					LevelID:        1,
					GroupID:        "A",
					Score:          cIdx * 5,
					TotalPoints:    100,
					CorrectCount:   cIdx,
					TotalQuestions: 20,
					SubmittedAt:    time.Now().UTC(),
				}

				<-startBarrier

				res, err := repo.SaveResult(ctx, input)
				outcomes[idx] = resultRecord{
					studentIdx: sIdx,
					returnedID: func() primitive.ObjectID {
						if res != nil {
							return res.ID
						}
						return primitive.NilObjectID
					}(),
					inputID: input.ID,
					score: func() int {
						if res != nil {
							return res.Score
						}
						return -1
					}(),
					err: err,
				}
			}(callIdx, s, c)
			callIdx++
		}
	}

	close(startBarrier)
	wg.Wait()

	// Analyze outcomes per student
	for s := 0; s < numStudents; s++ {
		var canonicalID primitive.ObjectID
		var canonicalScore int

		// First fetch the actual document saved in MongoDB
		dbResult, err := repo.GetStudentResult(ctx, quizID, studentIDs[s])
		if err != nil {
			t.Fatalf("student %d: GetStudentResult failed: %v", s, err)
		}
		if dbResult == nil {
			t.Fatalf("student %d: document missing in MongoDB!", s)
		}
		canonicalID = dbResult.ID
		canonicalScore = dbResult.Score

		// Verify every caller for this student got canonicalID and canonicalScore
		for _, out := range outcomes {
			if out.studentIdx != s {
				continue
			}
			if out.err != nil {
				t.Errorf("student %d: SaveResult returned error: %v", s, out.err)
				continue
			}
			if out.returnedID != canonicalID {
				t.Errorf("student %d: PHANTOM RETURN DETECTED! Expected %s, got %s",
					s, canonicalID.Hex(), out.returnedID.Hex())
			}
			if out.inputID != canonicalID {
				t.Errorf("student %d: INPUT NOT MUTATED! inputID=%s != canonicalID=%s",
					s, out.inputID.Hex(), canonicalID.Hex())
			}
			if out.score != canonicalScore {
				t.Errorf("student %d: DIVERGENT SCORE! Expected %d, got %d",
					s, canonicalScore, out.score)
			}
		}
	}

	// Verify exact document count in DB
	totalDocs, err := resultsCol.CountDocuments(ctx, bson.M{"quiz_id": quizID})
	if err != nil {
		t.Fatalf("CountDocuments failed: %v", err)
	}
	if int(totalDocs) != numStudents {
		t.Errorf("Expected %d documents in quiz_results, got %d", numStudents, totalDocs)
	}
}

// TestMilestone1_CohortFiltering_EdgeCases tests boundary and adversarial cases for GetLevelResults.
func TestMilestone1_CohortFiltering_EdgeCases(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo := repository.NewResultRepository(db)
	quizID := primitive.NewObjectID()

	// Seed records across diverse groups, levels, and scores
	sampleData := []struct {
		code    string
		levelID int
		groupID string
		score   int
	}{
		{"S1", 1, "A", 90},
		{"S2", 1, "A", 75},
		{"S3", 1, "B", 85},
		{"S4", 1, "C", 60},
		{"S5", 2, "A", 95},
		{"S6", 2, "B", 80},
		{"S7", 3, "ADV-1", 100}, // special characters in group ID
		{"S8", 3, "ADV.2", 88},  // dot in group ID
	}

	for _, d := range sampleData {
		res := &model.QuizResult{
			ID:          primitive.NewObjectID(),
			QuizID:      quizID,
			StudentID:   primitive.NewObjectID(),
			StudentCode: d.code,
			StudentName: "Name " + d.code,
			LevelID:     d.levelID,
			GroupID:     d.groupID,
			Score:       d.score,
			SubmittedAt: time.Now().UTC(),
		}
		if _, err := repo.SaveResult(ctx, res); err != nil {
			t.Fatalf("failed to seed %s: %v", d.code, err)
		}
	}

	testCases := []struct {
		name          string
		levelID       int
		groupID       string
		expectedCodes []string
	}{
		{
			name:          "Exact Level 1 Group A (scores 90, 75)",
			levelID:       1,
			groupID:       "A",
			expectedCodes: []string{"S1", "S2"},
		},
		{
			name:          "Trimmed Level 1 Group '  A  ' with leading/trailing spaces",
			levelID:       1,
			groupID:       "  A  ",
			expectedCodes: []string{"S1", "S2"},
		},
		{
			name:          "Trimmed Level 1 Group with tabs/newlines '\tB\n'",
			levelID:       1,
			groupID:       "\tB\n",
			expectedCodes: []string{"S3"},
		},
		{
			name:          "All groups in Level 1 (scores sorted: S1:90, S3:85, S2:75, S4:60)",
			levelID:       1,
			groupID:       "",
			expectedCodes: []string{"S1", "S3", "S2", "S4"},
		},
		{
			name:          "All groups in Level 1 with whitespace-only group string '   '",
			levelID:       1,
			groupID:       "   ",
			expectedCodes: []string{"S1", "S3", "S2", "S4"},
		},
		{
			name:          "All levels (levelID=0) for Group A (sorted: S5:95, S1:90, S2:75)",
			levelID:       0,
			groupID:       "A",
			expectedCodes: []string{"S5", "S1", "S2"},
		},
		{
			name:          "Negative levelID (levelID=-1) treated as any level",
			levelID:       -1,
			groupID:       "B",
			expectedCodes: []string{"S3", "S6"}, // S3:85, S6:80 (sorted DESC)
		},
		{
			name:          "Group with special characters hyphen 'ADV-1'",
			levelID:       3,
			groupID:       "ADV-1",
			expectedCodes: []string{"S7"},
		},
		{
			name:          "Group with special characters dot 'ADV.2'",
			levelID:       3,
			groupID:       "ADV.2",
			expectedCodes: []string{"S8"},
		},
		{
			name:          "Non-existent group returns empty slice without error",
			levelID:       1,
			groupID:       "NON_EXISTENT",
			expectedCodes: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := repo.GetLevelResults(ctx, quizID, tc.levelID, tc.groupID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != len(tc.expectedCodes) {
				t.Fatalf("expected %d results, got %d", len(tc.expectedCodes), len(results))
			}

			// Verify ordering and codes
			for i, expCode := range tc.expectedCodes {
				if results[i].StudentCode != expCode {
					t.Errorf("at index %d: expected student %s, got %s (score %d)",
						i, expCode, results[i].StudentCode, results[i].Score)
				}
			}

			// Verify descending score invariant
			for i := 1; i < len(results); i++ {
				if results[i].Score > results[i-1].Score {
					t.Errorf("scores not sorted DESC: result[%d]=%d > result[%d]=%d",
						i, results[i].Score, i-1, results[i-1].Score)
				}
			}
		})
	}
}
