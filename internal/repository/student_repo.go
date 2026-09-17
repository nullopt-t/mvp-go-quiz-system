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

type CohortCount struct {
	LevelID int    `bson:"level_id" json:"level_id"`
	GroupID string `bson:"group_id" json:"group_id"`
	Count   int64  `bson:"count" json:"count"`
}

type StudentRepository interface {
	FindByCode(ctx context.Context, code string) (*model.Student, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*model.Student, error)
	Count(ctx context.Context) (int64, error)
	CountByLevelAndGroup(ctx context.Context, levelID int, groupID string) (int64, error)
	GetCohortDistribution(ctx context.Context) ([]CohortCount, error)
	CreateOrUpdate(ctx context.Context, student *model.Student) error
	BulkInsert(ctx context.Context, students []model.Student) error
	BulkUpsert(ctx context.Context, students []model.Student) (int, error)
	GetAll(ctx context.Context, levelID int, groupID string, limit, offset int64) ([]model.Student, error)
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

func (r *studentRepository) CountByLevelAndGroup(ctx context.Context, levelID int, groupID string) (int64, error) {
	filter := bson.M{}
	if levelID > 0 {
		filter["level_id"] = levelID
	}
	if groupID != "" {
		filter["group_id"] = groupID
	}
	return r.col.CountDocuments(ctx, filter)
}

func (r *studentRepository) GetCohortDistribution(ctx context.Context) ([]CohortCount, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"is_active": true}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"level_id": "$level_id",
				"group_id": "$group_id",
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":      0,
			"level_id": "$_id.level_id",
			"group_id": "$_id.group_id",
			"count":    "$count",
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "level_id", Value: 1},
			{Key: "group_id", Value: 1},
		}}},
	}

	cursor, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate cohort distribution: %w", err)
	}
	defer cursor.Close(ctx)

	var results []CohortCount
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode cohort distribution: %w", err)
	}
	return results, nil
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

func (r *studentRepository) BulkUpsert(ctx context.Context, students []model.Student) (int, error) {
	if len(students) == 0 {
		return 0, nil
	}

	total := 0
	batchSize := 1000
	now := time.Now().UTC()

	for i := 0; i < len(students); i += batchSize {
		end := i + batchSize
		if end > len(students) {
			end = len(students)
		}

		batch := students[i:end]
		models := make([]mongo.WriteModel, len(batch))

		for idx, s := range batch {
			filter := bson.M{"student_code": s.StudentCode}
			update := bson.M{
				"$set": bson.M{
					"name":       s.Name,
					"level_id":   s.LevelID,
					"group_id":   s.GroupID,
					"is_active":  s.IsActive,
					"updated_at": now,
				},
				"$setOnInsert": bson.M{
					"_id":        primitive.NewObjectID(),
					"created_at": now,
				},
			}
			models[idx] = mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
		}

		opts := options.BulkWrite().SetOrdered(false)
		res, err := r.col.BulkWrite(ctx, models, opts)
		if err != nil {
			return total, fmt.Errorf("bulk upsert error: %w", err)
		}
		total += int(res.UpsertedCount + res.ModifiedCount + res.MatchedCount)
	}

	return total, nil
}

func (r *studentRepository) GetAll(ctx context.Context, levelID int, groupID string, limit, offset int64) ([]model.Student, error) {
	filter := bson.M{}
	if levelID > 0 {
		filter["level_id"] = levelID
	}
	if groupID != "" {
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

func (r *studentRepository) CreateOrUpdate(ctx context.Context, student *model.Student) error {
	now := time.Now().UTC()
	if student.ID.IsZero() {
		student.ID = primitive.NewObjectID()
	}
	if student.CreatedAt.IsZero() {
		student.CreatedAt = now
	}

	filter := bson.M{"student_code": student.StudentCode}
	update := bson.M{
		"$set": bson.M{
			"name":       student.Name,
			"level_id":   student.LevelID,
			"group_id":   student.GroupID,
			"is_active":  student.IsActive,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"_id":        student.ID,
			"created_at": student.CreatedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save student: %w", err)
	}
	return nil
}
