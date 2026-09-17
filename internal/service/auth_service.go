package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"quiz-system/internal/config"
	"quiz-system/internal/model"
	"quiz-system/internal/repository"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials = errors.New("invalid student ID or credentials")
	ErrStudentInactive    = errors.New("student account is deactivated")
	ErrInvalidToken       = errors.New("invalid or expired authentication token")
)

type AuthService interface {
	LoginStudent(ctx context.Context, studentCode string) (string, *model.Student, error)
	LoginAdmin(pin string) (string, error)
	ValidateToken(tokenString string) (*model.AuthClaims, error)
}

type authService struct {
	studentRepo repository.StudentRepository
	cfg         *config.Config
}

func NewAuthService(studentRepo repository.StudentRepository, cfg *config.Config) AuthService {
	return &authService{
		studentRepo: studentRepo,
		cfg:         cfg,
	}
}

func (s *authService) LoginStudent(ctx context.Context, studentCode string) (string, *model.Student, error) {
	student, err := s.studentRepo.FindByCode(ctx, studentCode)
	if err != nil {
		return "", nil, err
	}
	if student == nil {
		return "", nil, ErrInvalidCredentials
	}
	if !student.IsActive {
		return "", nil, ErrStudentInactive
	}

	claims := &model.AuthClaims{
		StudentID:   student.ID.Hex(),
		StudentCode: student.StudentCode,
		Name:        student.Name,
		LevelID:     student.LevelID,
		GroupID:     student.GroupID,
		Role:        "student",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "college-quiz-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.cfg.JWTSecret)
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign jwt: %w", err)
	}

	return tokenStr, student, nil
}

func (s *authService) LoginAdmin(pin string) (string, error) {
	if pin != s.cfg.AdminPIN {
		return "", ErrInvalidCredentials
	}

	claims := &model.AuthClaims{
		Role: "admin",
		Name: "College Staff / Administrator",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(4 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "college-quiz-system",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.cfg.JWTSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign admin jwt: %w", err)
	}

	return tokenStr, nil
}

func (s *authService) ValidateToken(tokenString string) (*model.AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.AuthClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.cfg.JWTSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*model.AuthClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
