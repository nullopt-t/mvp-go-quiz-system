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
	GetAllSorted(ctx context.Context, levelID int, groupID string, sortBy string, sortOrder int, limit, offset int64) ([]model.Student, error)
	GetByCursor(ctx context.Context, levelID int, groupID string, sortBy string, sortOrder int, cursorVal string, cursorID primitive.ObjectID, direction string, limit int64) ([]model.Student, bool, error)
	SetActive(ctx context.Context, id primitive.ObjectID, isActive bool) error
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
	return r.GetAllSorted(ctx, levelID, groupID, "student_code", 1, limit, offset)
}

func (r *studentRepository) GetAllSorted(ctx context.Context, levelID int, groupID string, sortBy string, sortOrder int, limit, offset int64) ([]model.Student, error) {
	filter := bson.M{}
	if levelID > 0 {
		filter["level_id"] = levelID
	}
	if groupID != "" {
		filter["group_id"] = groupID
	}

	validSortFields := map[string]string{
		"code":       "student_code",
		"name":       "name",
		"level":      "level_id",
		"group":      "group_id",
		"created_at": "created_at",
		"time":       "created_at",
	}

	sortField, ok := validSortFields[sortBy]
	if !ok {
		sortField = "student_code"
	}
	if sortOrder != 1 && sortOrder != -1 {
		sortOrder = 1
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortOrder}}).
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

func (r *studentRepository) GetByCursor(
	ctx context.Context,
	levelID int,
	groupID string,
	sortBy string,
	sortOrder int,
	cursorVal string,
	cursorID primitive.ObjectID,
	direction string,
	limit int64,
) ([]model.Student, bool, error) {
	filter := bson.M{}
	if levelID > 0 {
		filter["level_id"] = levelID
	}
	if groupID != "" {
		filter["group_id"] = groupID
	}

	validSortFields := map[string]string{
		"code":       "student_code",
		"name":       "name",
		"level":      "level_id",
		"group":      "group_id",
		"created_at": "created_at",
		"time":       "created_at",
	}

	sortField, ok := validSortFields[sortBy]
	if !ok {
		sortField = "student_code"
	}
	if sortOrder != 1 && sortOrder != -1 {
		sortOrder = 1
	}

	// Determine operator based on sort order and direction
	// If direction == "next" and sortOrder == 1 (ASC) -> field > cursorVal OR (field == cursorVal AND _id > cursorID)
	// If direction == "prev" and sortOrder == 1 (ASC) -> field < cursorVal OR (field == cursorVal AND _id < cursorID)
	isForward := direction != "prev"
	actualSortOrder := sortOrder
	if !isForward {
		// Reverse sort order when moving backwards to fetch closest preceding items
		actualSortOrder = -sortOrder
	}

	if !cursorID.IsZero() {
		gtLtOp := "$gt"
		if (!isForward && sortOrder == 1) || (isForward && sortOrder == -1) {
			gtLtOp = "$lt"
		}

		if sortField == "created_at" {
			var cursorTime time.Time
			if t, err := time.Parse(time.RFC3339Nano, cursorVal); err == nil {
				cursorTime = t
			} else if t, err := time.Parse(time.RFC3339, cursorVal); err == nil {
				cursorTime = t
			}
			filter["$or"] = []bson.M{
				{sortField: bson.M{gtLtOp: cursorTime}},
				{sortField: cursorTime, "_id": bson.M{gtLtOp: cursorID}},
			}
		} else {
			filter["$or"] = []bson.M{
				{sortField: bson.M{gtLtOp: cursorVal}},
				{sortField: cursorVal, "_id": bson.M{gtLtOp: cursorID}},
			}
		}
	}

	// Request limit + 1 to check whether there is a next/more page
	findOpts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: actualSortOrder}, {Key: "_id", Value: actualSortOrder}}).
		SetLimit(limit + 1)

	cursor, err := r.col.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, false, fmt.Errorf("failed to execute cursor query: %w", err)
	}
	defer cursor.Close(ctx)

	var items []model.Student
	if err := cursor.All(ctx, &items); err != nil {
		return nil, false, fmt.Errorf("failed to decode cursor items: %w", err)
	}

	hasMore := int64(len(items)) > limit
	if hasMore {
		items = items[:limit]
	}

	// If navigating backwards, reverse the slice back to normal display order
	if !isForward {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}

	return items, hasMore, nil
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

func (r *studentRepository) SetActive(ctx context.Context, id primitive.ObjectID, isActive bool) error {
	now := time.Now().UTC()
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"is_active":  isActive,
			"updated_at": now,
		},
	}
	_, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update student activation: %w", err)
	}
	return nil
}

