package service

import (
	"context"
	"fmt"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ResultService interface {
	CalculateAndSubmit(ctx context.Context, quizID, studentID primitive.ObjectID, studentCode, studentName string, levelID, groupID int) (*model.QuizResult, error)
	GetStudentResult(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizResult, error)
	GetStudentHistory(ctx context.Context, studentID primitive.ObjectID) ([]model.QuizResult, error)
	GetQuizLeaderboard(ctx context.Context, quizID primitive.ObjectID) ([]model.QuizResult, error)
	GetGroupAnalytics(ctx context.Context, quizID primitive.ObjectID, levelID, groupID int) ([]model.QuizResult, error)
}

type resultService struct {
	resultRepo  repository.ResultRepository
	answerRepo  repository.AnswerRepository
	quizRepo    repository.QuizRepository
	studentRepo repository.StudentRepository
}

func NewResultService(
	resultRepo repository.ResultRepository,
	answerRepo repository.AnswerRepository,
	quizRepo repository.QuizRepository,
	studentRepo repository.StudentRepository,
) ResultService {
	return &resultService{
		resultRepo:  resultRepo,
		answerRepo:  answerRepo,
		quizRepo:    quizRepo,
		studentRepo: studentRepo,
	}
}

func (s *resultService) CalculateAndSubmit(ctx context.Context, quizID, studentID primitive.ObjectID, studentCode, studentName string, levelID, groupID int) (*model.QuizResult, error) {
	// Check if already calculated
	existing, err := s.resultRepo.GetStudentResult(ctx, quizID, studentID)
	if err == nil && existing != nil {
		return existing, nil
	}

	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quiz: %w", err)
	}
	if quiz == nil {
		return nil, ErrQuizNotFound
	}

	// Fetch latest student answers from append-only log
	latestAnswers, err := s.answerRepo.GetLatestAnswersMap(ctx, quizID, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get answers: %w", err)
	}

	// Build correct answer lookup map: QuestionID -> CorrectOptionID
	correctAnswersMap := make(map[int]int)
	questionPointsMap := make(map[int]int)
	totalPossiblePoints := 0

	for _, q := range quiz.Questions {
		totalPossiblePoints += q.Points
		questionPointsMap[q.ID] = q.Points
		for _, opt := range q.Options {
			if opt.IsCorrect {
				correctAnswersMap[q.ID] = opt.ID
				break
			}
		}
	}

	// Calculate score
	score := 0
	correctCount := 0
	for qID, studentAnsID := range latestAnswers {
		correctAnsID, hasCorrect := correctAnswersMap[qID]
		if hasCorrect && studentAnsID == correctAnsID {
			score += questionPointsMap[qID]
			correctCount++
		}
	}

	result := &model.QuizResult{
		ID:             primitive.NewObjectID(),
		QuizID:         quizID,
		StudentID:      studentID,
		StudentCode:    studentCode,
		StudentName:    studentName,
		LevelID:        levelID,
		GroupID:        groupID,
		Score:          score,
		TotalPoints:    totalPossiblePoints,
		CorrectCount:   correctCount,
		TotalQuestions: len(quiz.Questions),
		SubmittedAt:    time.Now().UTC(),
	}

	if err := s.resultRepo.SaveResult(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to save result: %w", err)
	}

	return result, nil
}

func (s *resultService) GetStudentResult(ctx context.Context, quizID, studentID primitive.ObjectID) (*model.QuizResult, error) {
	return s.resultRepo.GetStudentResult(ctx, quizID, studentID)
}

func (s *resultService) GetStudentHistory(ctx context.Context, studentID primitive.ObjectID) ([]model.QuizResult, error) {
	return s.resultRepo.GetStudentResults(ctx, studentID)
}

func (s *resultService) GetQuizLeaderboard(ctx context.Context, quizID primitive.ObjectID) ([]model.QuizResult, error) {
	return s.resultRepo.GetQuizResults(ctx, quizID)
}

func (s *resultService) GetGroupAnalytics(ctx context.Context, quizID primitive.ObjectID, levelID, groupID int) ([]model.QuizResult, error) {
	return s.resultRepo.GetLevelResults(ctx, quizID, levelID, groupID)
}
