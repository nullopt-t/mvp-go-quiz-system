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

type ResultRepository interface {
	SaveResult(ctx context.Context, result *model.QuizResult) error
	GetStudentResult(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizResult, error)
	GetStudentResults(ctx context.Context, studentID primitive.ObjectID) ([]model.QuizResult, error)
	GetQuizResults(ctx context.Context, quizID primitive.ObjectID) ([]model.QuizResult, error)
	GetLevelResults(ctx context.Context, quizID primitive.ObjectID, levelID, groupID int) ([]model.QuizResult, error)
}

type resultRepository struct {
	col *mongo.Collection
}

func NewResultRepository(db *mongo.Database) ResultRepository {
	return &resultRepository{
		col: db.Collection("quiz_results"),
	}
}

func (r *resultRepository) SaveResult(ctx context.Context, result *model.QuizResult) error {
	if result.ID.IsZero() {
		result.ID = primitive.NewObjectID()
	}
	if result.SubmittedAt.IsZero() {
		result.SubmittedAt = time.Now().UTC()
	}

	filter := bson.M{
		"quiz_id":    result.QuizID,
		"student_id": result.StudentID,
	}

	update := bson.M{
		"$setOnInsert": result,
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save result: %w", err)
	}
	return nil
}

func (r *resultRepository) GetStudentResult(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizResult, error) {
	var result model.QuizResult
	err := r.col.FindOne(ctx, bson.M{"quiz_id": quizID, "student_id": studentID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch student result: %w", err)
	}
	return &result, nil
}

func (r *resultRepository) GetStudentResults(ctx context.Context, studentID primitive.ObjectID) ([]model.QuizResult, error) {
	opts := options.Find().SetSort(bson.D{{Key: "submitted_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{"student_id": studentID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query student results: %w", err)
	}
	defer cursor.Close(ctx)

	var results []model.QuizResult
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}
	return results, nil
}

func (r *resultRepository) GetQuizResults(ctx context.Context, quizID primitive.ObjectID) ([]model.QuizResult, error) {
	opts := options.Find().SetSort(bson.D{{Key: "score", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{"quiz_id": quizID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query quiz results: %w", err)
	}
	defer cursor.Close(ctx)

	var results []model.QuizResult
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode quiz results: %w", err)
	}
	return results, nil
}

func (r *resultRepository) GetLevelResults(ctx context.Context, quizID primitive.ObjectID, levelID, groupID int) ([]model.QuizResult, error) {
	filter := bson.M{"quiz_id": quizID}
	if levelID > 0 {
		filter["level_id"] = levelID
	}
	if groupID > 0 {
		filter["group_id"] = groupID
	}

	opts := options.Find().SetSort(bson.D{{Key: "score", Value: -1}})
	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query level results: %w", err)
	}
	defer cursor.Close(ctx)

	var results []model.QuizResult
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode level results: %w", err)
	}
	return results, nil
}
