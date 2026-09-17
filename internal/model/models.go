package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Level and Group hierarchy definitions
type GroupInfo struct {
	LevelID  int `bson:"level_id" json:"level_id"`
	GroupID  int `bson:"group_id" json:"group_id"`
	Capacity int `bson:"capacity" json:"capacity"`
}

// Student represents a registered student in the college
type Student struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentCode string             `bson:"student_code" json:"student_code"` // Login ID (e.g. STU-1001)
	Name        string             `bson:"name" json:"name"`
	LevelID     int                `bson:"level_id" json:"level_id"` // 1 to 5
	GroupID     int                `bson:"group_id" json:"group_id"` // 1 to 4
	IsActive    bool               `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// Option represents an answer option for a question
type Option struct {
	ID        int    `bson:"id" json:"id"`
	Text      string `bson:"text" json:"text"`
	IsCorrect bool   `bson:"is_correct" json:"is_correct"`
}

// Question represents a question in a quiz
type Question struct {
	ID      int      `bson:"id" json:"id"`
	Text    string   `bson:"text" json:"text"`
	Points  int      `bson:"points" json:"points"`
	Options []Option `bson:"options" json:"options"`
}

// Quiz represents a scheduled quiz
type Quiz struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title           string             `bson:"title" json:"title"`
	Description     string             `bson:"description" json:"description"`
	LevelID         int                `bson:"level_id" json:"level_id"`       // 1 to 5 (or 0 for all)
	GroupIDs        []int              `bson:"group_ids" json:"group_ids"`     // 1 to 4 (empty means all groups in level)
	DurationMinutes int                `bson:"duration_minutes" json:"duration_minutes"`
	StartTime       time.Time          `bson:"start_time" json:"start_time"`
	EndTime         time.Time          `bson:"end_time" json:"end_time"`
	IsActive        bool               `bson:"is_active" json:"is_active"`
	Questions       []Question         `bson:"questions" json:"questions"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
}

// StudentAnswer is an APPEND-ONLY document logged every time an answer is submitted via Next button
type StudentAnswer struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	QuizID      primitive.ObjectID `bson:"quiz_id" json:"quiz_id"`
	StudentID   primitive.ObjectID `bson:"student_id" json:"student_id"`
	StudentCode string             `bson:"student_code" json:"student_code"`
	QuestionID  int                `bson:"question_id" json:"question_id"`
	AnswerID    int                `bson:"answer_id" json:"answer_id"`
	SubmittedAt time.Time          `bson:"submitted_at" json:"submitted_at"`
}

// QuizResult stores the calculated final score for a student once quiz is finished/submitted
type QuizResult struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	QuizID         primitive.ObjectID `bson:"quiz_id" json:"quiz_id"`
	StudentID      primitive.ObjectID `bson:"student_id" json:"student_id"`
	StudentCode    string             `bson:"student_code" json:"student_code"`
	StudentName    string             `bson:"student_name" json:"student_name"`
	LevelID        int                `bson:"level_id" json:"level_id"`
	GroupID        int                `bson:"group_id" json:"group_id"`
	Score          int                `bson:"score" json:"score"`
	TotalPoints    int                `bson:"total_points" json:"total_points"`
	CorrectCount   int                `bson:"correct_count" json:"correct_count"`
	TotalQuestions int                `bson:"total_questions" json:"total_questions"`
	SubmittedAt    time.Time          `bson:"submitted_at" json:"submitted_at"`
}

// QuizSessionRecord tracks when a student started a quiz to enforce duration
type QuizSessionRecord struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	QuizID    primitive.ObjectID `bson:"quiz_id" json:"quiz_id"`
	StudentID primitive.ObjectID `bson:"student_id" json:"student_id"`
	StartedAt time.Time          `bson:"started_at" json:"started_at"`
}

// JWT Claims
type AuthClaims struct {
	StudentID   string `json:"student_id,omitempty"`
	StudentCode string `json:"student_code,omitempty"`
	Name        string `json:"name,omitempty"`
	LevelID     int    `json:"level_id,omitempty"`
	GroupID     int    `json:"group_id,omitempty"`
	Role        string `json:"role"` // "student" or "admin"
	jwt.RegisteredClaims
}
