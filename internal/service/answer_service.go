package service

import (
	"context"
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

	// F-05: Reject if quiz is not active (admin may have deactivated it)
	if !quiz.IsActive {
		return ErrQuizInactive
	}

	now := time.Now().UTC()
	quizEnd := quiz.StartTime.Add(time.Duration(quiz.DurationMinutes) * time.Minute)

	// F-02: Reject answers submitted before the quiz has started
	if now.Before(quiz.StartTime) {
		return ErrQuizNotStarted
	}
	// Add 30 seconds grace period for network latency
	if now.After(quizEnd.Add(30 * time.Second)) {
		return ErrQuizExpired
	}

	// F-09: Validate that questionID and answerID belong to this quiz
	validQuestion := false
	validAnswer := false
	for _, q := range quiz.Questions {
		if q.ID == questionID {
			validQuestion = true
			for _, opt := range q.Options {
				if opt.ID == answerID {
					validAnswer = true
					break
				}
			}
			break
		}
	}
	if !validQuestion || !validAnswer {
		return ErrInvalidQuestionOrAnswer
	}

	// Append record (No SQL UPDATE, pure INSERT)
	answerDoc := &model.StudentAnswer{
		ID:          primitive.NewObjectID(),
		QuizID:      quizID,
		StudentID:   studentID,
		StudentCode: studentCode,
		QuestionID:  questionID,
		AnswerID:    answerID,
		SubmittedAt: now,
	}

	return s.answerRepo.AppendAnswer(ctx, answerDoc)
}

func (s *answerService) GetStudentAnswerState(ctx context.Context, quizID, studentID primitive.ObjectID) (map[int]int, error) {
	return s.answerRepo.GetLatestAnswersMap(ctx, quizID, studentID)
}
