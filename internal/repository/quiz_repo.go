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

type QuizRepository interface {
	Create(ctx context.Context, quiz *model.Quiz) error
	Update(ctx context.Context, quiz *model.Quiz) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.Quiz, error)
	GetAll(ctx context.Context) ([]model.Quiz, error)
	GetAvailableForStudent(ctx context.Context, levelID int, groupID string) ([]model.Quiz, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
}

type quizRepository struct {
	col *mongo.Collection
}

func NewQuizRepository(db *mongo.Database) QuizRepository {
	return &quizRepository{
		col: db.Collection("quizzes"),
	}
}

func (r *quizRepository) Create(ctx context.Context, quiz *model.Quiz) error {
	if quiz.ID.IsZero() {
		quiz.ID = primitive.NewObjectID()
	}
	if quiz.CreatedAt.IsZero() {
		quiz.CreatedAt = time.Now().UTC()
	}

	_, err := r.col.InsertOne(ctx, quiz)
	if err != nil {
		return fmt.Errorf("failed to insert quiz: %w", err)
	}
	return nil
}

func (r *quizRepository) Update(ctx context.Context, quiz *model.Quiz) error {
	filter := bson.M{"_id": quiz.ID}
	update := bson.M{
		"$set": bson.M{
			"title":            quiz.Title,
			"description":      quiz.Description,
			"level_id":         quiz.LevelID,
			"group_ids":        quiz.GroupIDs,
			"duration_minutes": quiz.DurationMinutes,
			"start_time":       quiz.StartTime,
			"end_time":         quiz.EndTime,
			"is_active":        quiz.IsActive,
			"questions":        quiz.Questions,
		},
	}

	_, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update quiz: %w", err)
	}
	return nil
}

func (r *quizRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Quiz, error) {
	var quiz model.Quiz
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&quiz)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get quiz: %w", err)
	}
	return &quiz, nil
}

func (r *quizRepository) GetAll(ctx context.Context) ([]model.Quiz, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list quizzes: %w", err)
	}
	defer cursor.Close(ctx)

	var quizzes []model.Quiz
	if err := cursor.All(ctx, &quizzes); err != nil {
		return nil, fmt.Errorf("failed to decode quizzes: %w", err)
	}
	return quizzes, nil
}

func (r *quizRepository) GetAvailableForStudent(ctx context.Context, levelID int, groupID string) ([]model.Quiz, error) {
	groupFilter := bson.M{"$size": 0}
	if groupID != "" {
		groupFilter = bson.M{
			"$or": []bson.M{
				{"group_ids": bson.M{"$size": 0}},
				{"group_ids": groupID},
			},
		}
	}

	filter := bson.M{
		"is_active": true,
		"$and": []bson.M{
			{
				"$or": []bson.M{
					{"level_id": 0},
					{"level_id": levelID},
				},
			},
			groupFilter,
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get available quizzes: %w", err)
	}
	defer cursor.Close(ctx)

	var quizzes []model.Quiz
	if err := cursor.All(ctx, &quizzes); err != nil {
		return nil, fmt.Errorf("failed to decode available quizzes: %w", err)
	}
	return quizzes, nil
}

func (r *quizRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
