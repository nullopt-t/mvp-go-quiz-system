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

type StudentRepository interface {
	FindByCode(ctx context.Context, code string) (*model.Student, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*model.Student, error)
	Count(ctx context.Context) (int64, error)
	CountByLevelAndGroup(ctx context.Context, levelID, groupID int) (int64, error)
	BulkInsert(ctx context.Context, students []model.Student) error
	GetAll(ctx context.Context, levelID, groupID int, limit, offset int64) ([]model.Student, error)
}

type studentRepository struct {
	col *mongo.Collection
}

func NewStudentRepository(db *mongo.Database) StudentRepository {
	return &studentRepository{
		col: db.Collection("students"),
	}
}

func (r *studentRepository) FindByCode(ctx context.Context, code string) (*model.Student, error) {
	var student model.Student
	err := r.col.FindOne(ctx, bson.M{"student_code": code}).Decode(&student)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query student by code: %w", err)
	}
	return &student, nil
}

func (r *studentRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Student, error) {
	var student model.Student
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&student)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query student by id: %w", err)
	}
	return &student, nil
}

func (r *studentRepository) Count(ctx context.Context) (int64, error) {
	return r.col.CountDocuments(ctx, bson.M{})
}

func (r *studentRepository) CountByLevelAndGroup(ctx context.Context, levelID, groupID int) (int64, error) {
	filter := bson.M{}
	if levelID > 0 {
		filter["level_id"] = levelID
	}
	if groupID > 0 {
		filter["group_id"] = groupID
	}
	return r.col.CountDocuments(ctx, filter)
}

func (r *studentRepository) BulkInsert(ctx context.Context, students []model.Student) error {
	if len(students) == 0 {
		return nil
	}

	docs := make([]interface{}, len(students))
	now := time.Now().UTC()
	for i, s := range students {
		if s.ID.IsZero() {
			s.ID = primitive.NewObjectID()
		}
		if s.CreatedAt.IsZero() {
			s.CreatedAt = now
		}
		docs[i] = s
	}

	opts := options.InsertMany().SetOrdered(false)
	_, err := r.col.InsertMany(ctx, docs, opts)
	if err != nil {
		return fmt.Errorf("failed to bulk insert students: %w", err)
	}
	return nil
}

func (r *studentRepository) GetAll(ctx context.Context, levelID, groupID int, limit, offset int64) ([]model.Student, error) {
	filter := bson.M{}
	if levelID > 0 {
		filter["level_id"] = levelID
	}
	if groupID > 0 {
		filter["group_id"] = groupID
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: "student_code", Value: 1}}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := r.col.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list students: %w", err)
	}
	defer cursor.Close(ctx)

	var students []model.Student
	if err := cursor.All(ctx, &students); err != nil {
		return nil, fmt.Errorf("failed to decode students: %w", err)
	}
	return students, nil
}
