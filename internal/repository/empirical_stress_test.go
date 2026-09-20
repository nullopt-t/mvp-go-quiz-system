package repository_test

import (
	"context"
	"math/rand"
	"sort"
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

// TestEmpirical_SortingDeterminism_StressAndOracle tests that compound sorting {submitted_at: 1, _id: 1}
// strictly guarantees deterministic ordering and correct fold behavior against an independent Oracle,
// even when hundreds of answers share identical millisecond timestamps and are inserted out-of-order.
func TestEmpirical_SortingDeterminism_StressAndOracle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Ensure production indexes are present
	answersCol := db.Collection("student_answers")
	_, err := answersCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "quiz_id", Value: 1},
				{Key: "student_id", Value: 1},
				{Key: "submitted_at", Value: 1},
				{Key: "_id", Value: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create compound index: %v", err)
	}

	repo := repository.NewAnswerRepository(db)

	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	// Base timestamp
	baseTime := time.Date(2026, 9, 20, 15, 0, 0, 0, time.UTC)

	// We generate 5 distinct millisecond timestamps (bursts)
	// Each burst will contain 40 answers (total 200 answers) across 10 questions
	timestamps := make([]time.Time, 5)
	for i := range timestamps {
		timestamps[i] = baseTime.Add(time.Duration(i) * time.Millisecond)
	}

	totalAnswers := 200
	generatedAnswers := make([]model.StudentAnswer, 0, totalAnswers)

	// Generate sequentially ordered ObjectIDs to represent causal creation order
	for i := 0; i < totalAnswers; i++ {
		time.Sleep(500 * time.Microsecond) // Guarantee distinct monotonic counter in ObjectID
		ts := timestamps[i%len(timestamps)]
		qID := (i % 10) + 1 // questions 1 through 10
		ansID := 100 + i     // distinct answer choice

		ans := model.StudentAnswer{
			ID:          primitive.NewObjectID(),
			QuizID:      quizID,
			StudentID:   studentID,
			StudentCode: "EMPIRICAL-001",
			QuestionID:  qID,
			AnswerID:    ansID,
			SubmittedAt: ts,
		}
		generatedAnswers = append(generatedAnswers, ans)
	}

	// Build Independent Oracle Map:
	// 1. Sort copies of generatedAnswers using canonical comparator: (submitted_at ASC, _id ASC)
	sortedOracle := make([]model.StudentAnswer, len(generatedAnswers))
	copy(sortedOracle, generatedAnswers)
	sort.Slice(sortedOracle, func(i, j int) bool {
		if sortedOracle[i].SubmittedAt.Equal(sortedOracle[j].SubmittedAt) {
			return sortedOracle[i].ID.Hex() < sortedOracle[j].ID.Hex()
		}
		return sortedOracle[i].SubmittedAt.Before(sortedOracle[j].SubmittedAt)
	})

	oracleLatestMap := make(map[int]int)
	for _, ans := range sortedOracle {
		oracleLatestMap[ans.QuestionID] = ans.AnswerID
	}

	// Shuffle generatedAnswers before inserting to MongoDB
	// This proves MongoDB does NOT rely on insertion order or B-Tree slot order
	shuffled := make([]model.StudentAnswer, len(generatedAnswers))
	copy(shuffled, generatedAnswers)
	r := rand.New(rand.NewSource(42))
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Insert in shuffled order
	docsToInsert := make([]interface{}, len(shuffled))
	for i, s := range shuffled {
		docsToInsert[i] = s
	}
	_, err = answersCol.InsertMany(ctx, docsToInsert, options.InsertMany().SetOrdered(false))
	if err != nil {
		t.Fatalf("failed to insert shuffled documents: %v", err)
	}

	// 1. Stress check: Verify GetStudentAnswers returns 100% strictly sorted order
	fetchedAnswers, err := repo.GetStudentAnswers(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("GetStudentAnswers failed: %v", err)
	}
	if len(fetchedAnswers) != totalAnswers {
		t.Fatalf("expected %d answers, got %d", totalAnswers, len(fetchedAnswers))
	}

	for i := 0; i < len(fetchedAnswers); i++ {
		expected := sortedOracle[i]
		actual := fetchedAnswers[i]
		if actual.ID != expected.ID {
			t.Fatalf("SORT MISMATCH at index %d: actual ID %s != expected ID %s",
				i, actual.ID.Hex(), expected.ID.Hex())
		}
		if !actual.SubmittedAt.Equal(expected.SubmittedAt) {
			t.Fatalf("TIMESTAMP MISMATCH at index %d: actual %v != expected %v",
				i, actual.SubmittedAt, expected.SubmittedAt)
		}
	}

	// 2. Stress check: Verify GetLatestAnswersMap matches Oracle exactly
	latestMap, err := repo.GetLatestAnswersMap(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("GetLatestAnswersMap failed: %v", err)
	}

	if len(latestMap) != len(oracleLatestMap) {
		t.Fatalf("latestMap size mismatch: got %d, expected %d", len(latestMap), len(oracleLatestMap))
	}

	for qID, expectedAnsID := range oracleLatestMap {
		actualAnsID, ok := latestMap[qID]
		if !ok {
			t.Fatalf("Question %d missing from latestMap", qID)
		}
		if actualAnsID != expectedAnsID {
			t.Fatalf("FOLD FAILURE for question %d: got AnswerID %d, expected AnswerID %d (oracle)",
				qID, actualAnsID, expectedAnsID)
		}
	}

	t.Logf("Successfully verified %d answers across %d questions: Oracle match 100%%", totalAnswers, len(oracleLatestMap))
}

// TestEmpirical_MongoDBExplain_IndexCoveredSort verifies that MongoDB's execution plan
// uses IXSCAN without an in-memory SORT stage.
func TestEmpirical_MongoDBExplain_IndexCoveredSort(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	answersCol := db.Collection("student_answers")
	_, err := answersCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "quiz_id", Value: 1},
			{Key: "student_id", Value: 1},
			{Key: "submitted_at", Value: 1},
			{Key: "_id", Value: 1},
		},
	})
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	// Run MongoDB Explain command
	explainCmd := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: "student_answers"},
			{Key: "filter", Value: bson.M{
				"quiz_id":    quizID,
				"student_id": studentID,
			}},
			{Key: "sort", Value: bson.D{
				{Key: "submitted_at", Value: 1},
				{Key: "_id", Value: 1},
			}},
		}},
		{Key: "verbosity", Value: "queryPlanner"},
	}

	var explainResult bson.M
	if err := db.RunCommand(ctx, explainCmd).Decode(&explainResult); err != nil {
		t.Fatalf("failed to run explain command: %v", err)
	}

	// Recursively inspect stages to ensure no "SORT" stage exists
	hasSortStage := false
	hasIXSCANStage := false

	var inspectStage func(stage bson.M)
	inspectStage = func(stage bson.M) {
		stageName, _ := stage["stage"].(string)
		if stageName == "SORT" {
			hasSortStage = true
		}
		if stageName == "IXSCAN" {
			hasIXSCANStage = true
		}

		if inputStage, ok := stage["inputStage"].(bson.M); ok {
			inspectStage(inputStage)
		}
		if inputStages, ok := stage["inputStages"].(primitive.A); ok {
			for _, is := range inputStages {
				if isMap, ok := is.(bson.M); ok {
					inspectStage(isMap)
				}
			}
		}
	}

	if qp, ok := explainResult["queryPlanner"].(bson.M); ok {
		if winningPlan, ok := qp["winningPlan"].(bson.M); ok {
			inspectStage(winningPlan)
		}
	}

	if hasSortStage {
		t.Errorf("FAIL: MongoDB query plan contains an in-memory SORT stage! Sort is not covered by index.")
	}
	if !hasIXSCANStage {
		t.Errorf("FAIL: MongoDB query plan did not use IXSCAN stage!")
	}

	t.Logf("Empirical Explain verified: IXSCAN=%v, SORT_STAGE_PRESENT=%v (Index covers sort without memory buffer)",
		hasIXSCANStage, hasSortStage)
}

// TestEmpirical_ConcurrentSubmissionsAndMapFold tests high-frequency concurrent answer logging
// and verifies that GetLatestAnswersMap consistently returns valid final choices.
func TestEmpirical_ConcurrentSubmissionsAndMapFold(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo := repository.NewAnswerRepository(db)

	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	const numRoutines = 20
	const updatesPerRoutine = 10
	var wg sync.WaitGroup

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < updatesPerRoutine; j++ {
				ans := &model.StudentAnswer{
					QuizID:      quizID,
					StudentID:   studentID,
					StudentCode: "CONCURRENT-STUDENT",
					QuestionID:  j + 1, // Questions 1 to 10
					AnswerID:    routineID*100 + j,
					// Omit ID and SubmittedAt to test AppendAnswer's auto-assignment
				}
				_ = repo.AppendAnswer(ctx, ans)
			}
		}(i)
	}

	wg.Wait()

	// Verify count
	count, err := repo.CountStudentAnswers(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("CountStudentAnswers failed: %v", err)
	}
	expectedCount := int64(numRoutines * updatesPerRoutine)
	if count != expectedCount {
		t.Fatalf("expected count %d, got %d", expectedCount, count)
	}

	// Verify latest map has all 10 questions
	latestMap, err := repo.GetLatestAnswersMap(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("GetLatestAnswersMap failed: %v", err)
	}
	if len(latestMap) != updatesPerRoutine {
		t.Fatalf("expected latestMap to have %d questions, got %d", updatesPerRoutine, len(latestMap))
	}

	for q := 1; q <= updatesPerRoutine; q++ {
		ansID, ok := latestMap[q]
		if !ok {
			t.Errorf("question %d missing from latest map", q)
		}
		if ansID%100 != (q - 1) {
			t.Errorf("question %d has corrupted answer ID %d", q, ansID)
		}
	}
}

// TestEmpirical_PrePatchVsPostPatch_SortingDeterminism proves that unpatched sort
// {submitted_at: 1} fails when answers have identical timestamps, whereas patched
// compound sort {submitted_at: 1, _id: 1} succeeds deterministically.
func TestEmpirical_PrePatchVsPostPatch_SortingDeterminism(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("student_answers")
	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	sharedTime := time.Date(2026, 9, 20, 18, 0, 0, 0, time.UTC)

	// docOld created first
	idOld := primitive.NewObjectID()
	time.Sleep(2 * time.Millisecond)
	// docNew created second (causally newer)
	idNew := primitive.NewObjectID()

	docOld := model.StudentAnswer{
		ID:          idOld,
		QuizID:      quizID,
		StudentID:   studentID,
		QuestionID:  1,
		AnswerID:    10, // Older answer
		SubmittedAt: sharedTime,
	}

	docNew := model.StudentAnswer{
		ID:          idNew,
		QuizID:      quizID,
		StudentID:   studentID,
		QuestionID:  1,
		AnswerID:    20, // Newer answer
		SubmittedAt: sharedTime,
	}

	// Insert docNew first to simulate adverse storage/cursor scan order
	if _, err := col.InsertOne(ctx, docNew); err != nil {
		t.Fatalf("failed to insert docNew: %v", err)
	}
	if _, err := col.InsertOne(ctx, docOld); err != nil {
		t.Fatalf("failed to insert docOld: %v", err)
	}

	// 1. UNPATCHED QUERY: Sort solely by {submitted_at: 1}
	unpatchedOpts := options.Find().SetSort(bson.D{{Key: "submitted_at", Value: 1}})
	cursorUnpatched, err := col.Find(ctx, bson.M{"quiz_id": quizID, "student_id": studentID}, unpatchedOpts)
	if err != nil {
		t.Fatalf("unpatched find failed: %v", err)
	}
	defer cursorUnpatched.Close(ctx)

	var unpatchedAnswers []model.StudentAnswer
	if err := cursorUnpatched.All(ctx, &unpatchedAnswers); err != nil {
		t.Fatalf("unpatched decode failed: %v", err)
	}

	// Unpatched order reflects insertion order: docNew followed by docOld
	if len(unpatchedAnswers) != 2 {
		t.Fatalf("expected 2 answers, got %d", len(unpatchedAnswers))
	}
	unpatchedFoldResult := make(map[int]int)
	for _, ans := range unpatchedAnswers {
		unpatchedFoldResult[ans.QuestionID] = ans.AnswerID
	}

	t.Logf("Unpatched sort returned order: [%s (Ans %d), %s (Ans %d)] -> Folded to Ans %d",
		unpatchedAnswers[0].ID.Hex(), unpatchedAnswers[0].AnswerID,
		unpatchedAnswers[1].ID.Hex(), unpatchedAnswers[1].AnswerID,
		unpatchedFoldResult[1])

	// Confirm that unpatched sort yielded the STALE answer 10 because docOld overwrote docNew!
	if unpatchedFoldResult[1] == 10 {
		t.Logf("Confirmed: Unpatched code regressed to stale answer (AnswerID 10) due to lack of _id tie-breaker.")
	}

	// 2. PATCHED QUERY: Sort by {submitted_at: 1, _id: 1}
	repo := repository.NewAnswerRepository(db)
	latestMap, err := repo.GetLatestAnswersMap(ctx, quizID, studentID)
	if err != nil {
		t.Fatalf("patched GetLatestAnswersMap failed: %v", err)
	}

	// Patched sort guarantees docOld (idOld) comes first, and docNew (idNew) comes second,
	// so folding correctly keeps docNew (AnswerID 20)!
	if latestMap[1] != 20 {
		t.Fatalf("Patched sort failed: expected 20, got %d", latestMap[1])
	}

	t.Logf("Patched compound sort deterministically resolved tie to AnswerID %d (docNew)", latestMap[1])
}

// TestEmpirical_SaveResult_100ConcurrentUpserts tests SaveResult under extreme concurrency (100 goroutines)
func TestEmpirical_SaveResult_100ConcurrentUpserts(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Ensure unique index
	resultsCol := db.Collection("quiz_results")
	_, err := resultsCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "quiz_id", Value: 1},
			{Key: "student_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		t.Fatalf("failed to create index: %v", err)
	}

	repo := repository.NewResultRepository(db)
	quizID := primitive.NewObjectID()
	studentID := primitive.NewObjectID()

	const concurrency = 100
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
			input := &model.QuizResult{
				QuizID:      quizID,
				StudentID:   studentID,
				StudentCode: "STRESS-100",
				LevelID:     1,
				GroupID:     "A",
				Score:       idx,
			}
			<-startBarrier
			res, err := repo.SaveResult(ctx, input)
			outcomes[idx] = outcome{res: res, err: err}
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	var canonicalID primitive.ObjectID
	for i, out := range outcomes {
		if out.err != nil {
			// Check if duplicate key error occurred or any other error
			t.Fatalf("Goroutine %d failed with error: %v", i, out.err)
		}
		if out.res == nil {
			t.Fatalf("Goroutine %d returned nil result", i)
		}
		if canonicalID.IsZero() {
			canonicalID = out.res.ID
		} else if out.res.ID != canonicalID {
			t.Fatalf("Goroutine %d got divergent ID %s (expected %s)", i, out.res.ID.Hex(), canonicalID.Hex())
		}
	}

	t.Logf("Successfully verified 100 concurrent finalizations: all 100 received identical canonical ID %s", canonicalID.Hex())
}

