package repository

import (
	"context"
	"fmt"
	"time"

	"quiz-system/internal/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDatabase struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDatabase(cfg *config.Config) (*MongoDatabase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(cfg.MongoURI).
		SetMaxPoolSize(100).
		SetMinPoolSize(10).
		SetMaxConnIdleTime(5 * time.Minute)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	db := client.Database(cfg.DBName)
	mongoDb := &MongoDatabase{
		Client:   client,
		Database: db,
	}

	if err := mongoDb.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure indexes: %w", err)
	}

	return mongoDb, nil
}

func (db *MongoDatabase) ensureIndexes(ctx context.Context) error {
	// Students collection: indexes for unique identity and cursor-based seeks
	studentsCol := db.Database.Collection("students")
	_, err := studentsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "student_code", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		// Standalone sort & seek indexes (used when viewing all levels/groups)
		{
			Keys: bson.D{
				{Key: "student_code", Value: 1},
				{Key: "_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "name", Value: 1},
				{Key: "_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "created_at", Value: -1},
				{Key: "_id", Value: -1},
			},
		},
		// Filtered compound indexes (used when filtering by level and/or group)
		{
			Keys: bson.D{
				{Key: "level_id", Value: 1},
				{Key: "group_id", Value: 1},
				{Key: "student_code", Value: 1},
				{Key: "_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "level_id", Value: 1},
				{Key: "group_id", Value: 1},
				{Key: "name", Value: 1},
				{Key: "_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "level_id", Value: 1},
				{Key: "group_id", Value: 1},
				{Key: "created_at", Value: -1},
				{Key: "_id", Value: -1},
			},
		},
	})
	if err != nil {
		return err
	}

	// Student Answers collection (APPEND-ONLY): index on (quiz_id, student_id, question_id)
	answersCol := db.Database.Collection("student_answers")
	_, err = answersCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "quiz_id", Value: 1},
				{Key: "student_id", Value: 1},
				{Key: "submitted_at", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "quiz_id", Value: 1},
				{Key: "student_id", Value: 1},
				{Key: "question_id", Value: 1},
			},
		},
	})
	if err != nil {
		return err
	}

	// Quiz Results: unique on (quiz_id, student_id)
	resultsCol := db.Database.Collection("quiz_results")
	_, err = resultsCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "quiz_id", Value: 1},
			{Key: "student_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// Quiz Sessions: unique on (quiz_id, student_id)
	sessionsCol := db.Database.Collection("quiz_sessions")
	_, err = sessionsCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "quiz_id", Value: 1},
			{Key: "student_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	return nil
}
