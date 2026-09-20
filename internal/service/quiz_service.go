package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrQuizNotFound            = errors.New("quiz not found")
	ErrQuizInactive            = errors.New("quiz is currently not active")
	ErrQuizNotStarted          = errors.New("quiz has not started yet")
	ErrQuizExpired             = errors.New("quiz session has expired")
	ErrAlreadySubmitted        = errors.New("quiz has already been completed and submitted")
	ErrInvalidQuestionOrAnswer = errors.New("invalid question or answer ID for this quiz")
)

type StudentQuizView struct {
	Quiz             *model.Quiz
	RemainingSeconds int
	StartedAt        time.Time
	IsCompleted      bool
	PreviousAnswers  map[int]int // question_id -> answer_id
}

type QuizService interface {
	CreateQuiz(ctx context.Context, quiz *model.Quiz) error
	GetQuizByID(ctx context.Context, id primitive.ObjectID) (*model.Quiz, error)
	GetAllQuizzes(ctx context.Context) ([]model.Quiz, error)
	GetAvailableQuizzesForStudent(ctx context.Context, studentID primitive.ObjectID, levelID int, groupID string) ([]StudentQuizSummary, error)
	StartOrResumeQuiz(ctx context.Context, quizID, studentID primitive.ObjectID) (*StudentQuizView, error)
}

type StudentQuizSummary struct {
	Quiz        model.Quiz
	Status      string // "AVAILABLE", "IN_PROGRESS", "COMPLETED"
	Score       int
	TotalPoints int
}

type quizService struct {
	quizRepo    repository.QuizRepository
	sessionRepo repository.SessionRepository
	answerRepo  repository.AnswerRepository
	resultRepo  repository.ResultRepository
}

func NewQuizService(
	quizRepo repository.QuizRepository,
	sessionRepo repository.SessionRepository,
	answerRepo repository.AnswerRepository,
	resultRepo repository.ResultRepository,
) QuizService {
	return &quizService{
		quizRepo:    quizRepo,
		sessionRepo: sessionRepo,
		answerRepo:  answerRepo,
		resultRepo:  resultRepo,
	}
}

func (s *quizService) CreateQuiz(ctx context.Context, quiz *model.Quiz) error {
	if quiz.Title == "" {
		return errors.New("quiz title is required")
	}
	if quiz.DurationMinutes <= 0 {
		quiz.DurationMinutes = 30
	}
	return s.quizRepo.Create(ctx, quiz)
}

func (s *quizService) GetQuizByID(ctx context.Context, id primitive.ObjectID) (*model.Quiz, error) {
	return s.quizRepo.GetByID(ctx, id)
}

func (s *quizService) GetAllQuizzes(ctx context.Context) ([]model.Quiz, error) {
	return s.quizRepo.GetAll(ctx)
}

func (s *quizService) GetAvailableQuizzesForStudent(ctx context.Context, studentID primitive.ObjectID, levelID int, groupID string) ([]StudentQuizSummary, error) {
	quizzes, err := s.quizRepo.GetAvailableForStudent(ctx, levelID, groupID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	summaries := make([]StudentQuizSummary, 0, len(quizzes))
	for _, q := range quizzes {
		summary := StudentQuizSummary{
			Quiz:   q,
			Status: "AVAILABLE",
		}

		// Check if already completed
		res, err := s.resultRepo.GetStudentResult(ctx, q.ID, studentID)
		if err == nil && res != nil {
			summary.Status = "COMPLETED"
			summary.Score = res.Score
			summary.TotalPoints = res.TotalPoints
		} else {
			quizEnd := q.StartTime.Add(time.Duration(q.DurationMinutes) * time.Minute)
			if now.Before(q.StartTime) {
				summary.Status = "UPCOMING"
			} else if now.After(quizEnd) {
				// Ended/expired quiz without completion: do not display on student board
				continue
			} else {
				summary.Status = "LIVE"
			}
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (s *quizService) StartOrResumeQuiz(ctx context.Context, quizID, studentID primitive.ObjectID) (*StudentQuizView, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, err
	}
	if quiz == nil {
		return nil, ErrQuizNotFound
	}
	if !quiz.IsActive {
		return nil, ErrQuizInactive
	}

	now := time.Now().UTC()
	quizEnd := quiz.StartTime.Add(time.Duration(quiz.DurationMinutes) * time.Minute)

	if now.Before(quiz.StartTime) {
		return nil, fmt.Errorf("quiz will start at %s for all students", quiz.StartTime.Format("15:04:05 MST"))
	}

	// F-07 FIX: Check completion BEFORE expiration so finished students can always view results
	res, err := s.resultRepo.GetStudentResult(ctx, quizID, studentID)
	if err == nil && res != nil {
		return &StudentQuizView{
			Quiz:        quiz,
			IsCompleted: true,
		}, nil
	}

	if now.After(quizEnd) {
		return nil, ErrQuizExpired
	}

	// Synchronized remaining time for all students
	remaining := int(quizEnd.Sub(now).Seconds())
	if remaining < 0 {
		remaining = 0
	}

	// Get latest answers map to restore client state
	answersMap, err := s.answerRepo.GetLatestAnswersMap(ctx, quizID, studentID)
	if err != nil {
		answersMap = make(map[int]int)
	}

	// Sanitize quiz questions (strip is_correct)
	sanitizedQuiz := *quiz
	sanitizedQuestions := make([]model.Question, len(quiz.Questions))
	for i, q := range quiz.Questions {
		sanitizedOptions := make([]model.Option, len(q.Options))
		for j, opt := range q.Options {
			sanitizedOptions[j] = model.Option{
				ID:        opt.ID,
				Text:      opt.Text,
				IsCorrect: false, // Protected against inspection
			}
		}
		sanitizedQuestions[i] = model.Question{
			ID:      q.ID,
			Text:    q.Text,
			Points:  q.Points,
			Options: sanitizedOptions,
		}
	}
	sanitizedQuiz.Questions = sanitizedQuestions

	return &StudentQuizView{
		Quiz:             &sanitizedQuiz,
		RemainingSeconds: remaining,
		StartedAt:        quiz.StartTime,
		IsCompleted:      false,
		PreviousAnswers:  answersMap,
	}, nil
}
