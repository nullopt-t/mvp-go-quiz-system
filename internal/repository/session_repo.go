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

type SessionRepository interface {
	StartSessionIfAbsent(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizSessionRecord, error)
	GetSession(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizSessionRecord, error)
}

type sessionRepository struct {
	col *mongo.Collection
}

func NewSessionRepository(db *mongo.Database) SessionRepository {
	return &sessionRepository{
		col: db.Collection("quiz_sessions"),
	}
}

func (r *sessionRepository) StartSessionIfAbsent(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizSessionRecord, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"quiz_id":    quizID,
		"student_id": studentID,
	}

	update := bson.M{
		"$setOnInsert": bson.M{
			"quiz_id":    quizID,
			"student_id": studentID,
			"started_at": now,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var session model.QuizSessionRecord
	err := r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&session)
	if err != nil {
		return nil, fmt.Errorf("failed to get/set quiz session: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) GetSession(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizSessionRecord, error) {
	var session model.QuizSessionRecord
	err := r.col.FindOne(ctx, bson.M{"quiz_id": quizID, "student_id": studentID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}
