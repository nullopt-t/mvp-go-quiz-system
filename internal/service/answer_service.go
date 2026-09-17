package service

import (
	"context"
	"errors"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnswerService interface {
	RecordAnswer(ctx context.Context, quizID, studentID primitive.ObjectID, studentCode string, questionID, answerID int) error
	GetStudentAnswerState(ctx context.Context, quizID, studentID primitive.ObjectID) (map[int]int, error)
}

type answerService struct {
	answerRepo  repository.AnswerRepository
	quizRepo    repository.QuizRepository
	sessionRepo repository.SessionRepository
	resultRepo  repository.ResultRepository
}

func NewAnswerService(
	answerRepo repository.AnswerRepository,
	quizRepo repository.QuizRepository,
	sessionRepo repository.SessionRepository,
	resultRepo repository.ResultRepository,
) AnswerService {
	return &answerService{
		answerRepo:  answerRepo,
		quizRepo:    quizRepo,
		sessionRepo: sessionRepo,
		resultRepo:  resultRepo,
	}
}

func (s *answerService) RecordAnswer(ctx context.Context, quizID, studentID primitive.ObjectID, studentCode string, questionID, answerID int) error {
	// Check if already finalized
	existingResult, err := s.resultRepo.GetStudentResult(ctx, quizID, studentID)
	if err == nil && existingResult != nil {
		return ErrAlreadySubmitted
	}

	// Verify quiz exists and time limit
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil || quiz == nil {
		return ErrQuizNotFound
	}

	session, err := s.sessionRepo.GetSession(ctx, quizID, studentID)
	if err != nil || session == nil {
		return errors.New("quiz session not found")
	}

	allowedDuration := time.Duration(quiz.DurationMinutes) * time.Minute
	// Add 30 seconds grace period for network latency
	if time.Since(session.StartedAt) > (allowedDuration + 30*time.Second) {
		return ErrQuizExpired
	}

	// Append record (No SQL UPDATE, pure INSERT)
	answerDoc := &model.StudentAnswer{
		ID:          primitive.NewObjectID(),
		QuizID:      quizID,
		StudentID:   studentID,
		StudentCode: studentCode,
		QuestionID:  questionID,
		AnswerID:    answerID,
		SubmittedAt: time.Now().UTC(),
	}

	return s.answerRepo.AppendAnswer(ctx, answerDoc)
}

func (s *answerService) GetStudentAnswerState(ctx context.Context, quizID, studentID primitive.ObjectID) (map[int]int, error) {
	return s.answerRepo.GetLatestAnswersMap(ctx, quizID, studentID)
}
