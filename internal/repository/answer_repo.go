package repository

import (
	"context"
	"fmt"
	"time"

	"quiz-system/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AnswerRepository interface {
	// AppendAnswer strictly appends a new document for the given answer - no updates
	AppendAnswer(ctx context.Context, answer *model.StudentAnswer) error
	GetStudentAnswers(ctx context.Context, quizID, studentID primitive.ObjectID) ([]model.StudentAnswer, error)
	GetLatestAnswersMap(ctx context.Context, quizID, studentID primitive.ObjectID) (map[int]int, error)
	CountStudentAnswers(ctx context.Context, quizID, studentID primitive.ObjectID) (int64, error)
}

type answerRepository struct {
	col *mongo.Collection
}

func NewAnswerRepository(db *mongo.Database) AnswerRepository {
	return &answerRepository{
		col: db.Collection("student_answers"),
	}
}

// AppendAnswer performs an atomic append-only insert into the log collection
func (r *answerRepository) AppendAnswer(ctx context.Context, answer *model.StudentAnswer) error {
	if answer.ID.IsZero() {
		answer.ID = primitive.NewObjectID()
	}
	if answer.SubmittedAt.IsZero() {
		answer.SubmittedAt = time.Now().UTC()
	}

	_, err := r.col.InsertOne(ctx, answer)
	if err != nil {
		return fmt.Errorf("failed to append answer: %w", err)
	}
	return nil
}

// GetStudentAnswers retrieves all appended answer entries for a student on a specific quiz in chronological order
func (r *answerRepository) GetStudentAnswers(ctx context.Context, quizID, studentID primitive.ObjectID) ([]model.StudentAnswer, error) {
	filter := bson.M{
		"quiz_id":    quizID,
		"student_id": studentID,
	}

	opts := options.Find().SetSort(bson.D{
		{Key: "submitted_at", Value: 1},
		{Key: "_id", Value: 1},
	})
	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch student answers: %w", err)
	}
	defer cursor.Close(ctx)

	var answers []model.StudentAnswer
	if err := cursor.All(ctx, &answers); err != nil {
		return nil, fmt.Errorf("failed to decode student answers: %w", err)
	}
	return answers, nil
}

// GetLatestAnswersMap folds the append-only logs to get the most recent answer for each question
func (r *answerRepository) GetLatestAnswersMap(ctx context.Context, quizID, studentID primitive.ObjectID) (map[int]int, error) {
	answers, err := r.GetStudentAnswers(ctx, quizID, studentID)
	if err != nil {
		return nil, err
	}

	latestMap := make(map[int]int)
	for _, ans := range answers {
		latestMap[ans.QuestionID] = ans.AnswerID
	}
	return latestMap, nil
}

func (r *answerRepository) CountStudentAnswers(ctx context.Context, quizID, studentID primitive.ObjectID) (int64, error) {
	filter := bson.M{
		"quiz_id":    quizID,
		"student_id": studentID,
	}
	return r.col.CountDocuments(ctx, filter)
}
