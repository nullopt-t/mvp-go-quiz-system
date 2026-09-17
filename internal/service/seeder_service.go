package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SeederService interface {
	SeedInitialData(ctx context.Context) error
}

type seederService struct {
	studentRepo repository.StudentRepository
	quizRepo    repository.QuizRepository
}

func NewSeederService(studentRepo repository.StudentRepository, quizRepo repository.QuizRepository) SeederService {
	return &seederService{
		studentRepo: studentRepo,
		quizRepo:    quizRepo,
	}
}

func (s *seederService) SeedInitialData(ctx context.Context) error {
	count, err := s.studentRepo.Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to check student count: %w", err)
	}

	if count == 0 {
		csvPath := "data/test_students_25k.csv"
		if _, err := os.Stat(csvPath); os.IsNotExist(err) {
			csvPath = "/app/data/test_students_25k.csv"
		}
		if _, err := os.Stat(csvPath); os.IsNotExist(err) {
			csvPath = "test_students_25k.csv"
		}

		f, err := os.Open(csvPath)
		if err == nil {
			defer f.Close()
			log.Printf("Seeding students from %s...", csvPath)
			r := csv.NewReader(f)
			r.TrimLeadingSpace = true

			// Read header
			_, _ = r.Read()

			now := time.Now().UTC()
			batchSize := 1000
			batch := make([]model.Student, 0, batchSize)
			totalLoaded := 0

			for {
				record, err := r.Read()
				if err == io.EOF {
					break
				}
				if err != nil || len(record) < 4 {
					continue
				}

				code := strings.TrimSpace(record[0])
				name := strings.TrimSpace(record[1])
				lvl, _ := strconv.Atoi(strings.TrimSpace(record[2]))
				grp := strings.ToUpper(strings.TrimSpace(record[3]))

				if code == "" || name == "" {
					continue
				}
				if lvl < 1 || lvl > 5 {
					lvl = 1
				}
				if grp == "" {
					grp = "A"
				}

				student := model.Student{
					ID:          primitive.NewObjectID(),
					StudentCode: code,
					Name:        name,
					LevelID:     lvl,
					GroupID:     grp,
					IsActive:    true,
					CreatedAt:   now,
				}
				batch = append(batch, student)
				totalLoaded++

				if len(batch) >= batchSize {
					if err := s.studentRepo.BulkInsert(ctx, batch); err != nil {
						return fmt.Errorf("bulk insert failure: %w", err)
					}
					batch = batch[:0]
				}
			}

			if len(batch) > 0 {
				if err := s.studentRepo.BulkInsert(ctx, batch); err != nil {
					return fmt.Errorf("bulk insert failure: %w", err)
				}
			}
			log.Printf("Successfully seeded %d students from %s!", totalLoaded, csvPath)
		} else {
			log.Printf("Could not open 25k csv (%v), falling back to procedural seeder...", err)
			now := time.Now().UTC()
			batchSize := 1000
			batch := make([]model.Student, 0, batchSize)
			groups := []string{"A", "B", "C", "D"}

			for level := 1; level <= 5; level++ {
				for _, group := range groups {
					for stuNum := 1; stuNum <= 500; stuNum++ {
						code := fmt.Sprintf("L%d%s-%03d", level, group, stuNum)
						name := fmt.Sprintf("Student L%d-%s #%03d", level, group, stuNum)

						student := model.Student{
							ID:          primitive.NewObjectID(),
							StudentCode: code,
							Name:        name,
							LevelID:     level,
							GroupID:     group,
							IsActive:    true,
							CreatedAt:   now,
						}
						batch = append(batch, student)

						if len(batch) >= batchSize {
							if err := s.studentRepo.BulkInsert(ctx, batch); err != nil {
								return fmt.Errorf("bulk insert failure: %w", err)
							}
							batch = batch[:0]
						}
					}
				}
			}

			if len(batch) > 0 {
				if err := s.studentRepo.BulkInsert(ctx, batch); err != nil {
					return fmt.Errorf("bulk insert failure: %w", err)
				}
			}
		}
	}

	// Seed sample quizzes if none exist
	quizzes, err := s.quizRepo.GetAll(ctx)
	if err == nil && len(quizzes) == 0 {
		log.Println("Seeding sample quizzes...")
		now := time.Now().UTC()
		sampleQuizzes := []*model.Quiz{
			{
				ID:              primitive.NewObjectID(),
				Title:           "Midterm Assessment: Computer Science Fundamentals",
				Description:     "Synchronized exam covering algorithms, data structures, and computer architecture for all Level 1 students.",
				LevelID:         1,
				GroupIDs:        []string{}, // All groups in Level 1
				DurationMinutes: 45,
				StartTime:       now.Add(-5 * time.Minute), // Started 5 mins ago, live for next 40 mins
				EndTime:         now.Add(40 * time.Minute),
				IsActive:        true,
				CreatedAt:       now,
				Questions: []model.Question{
					{
						ID:     1,
						Text:   "What is the average time complexity of searching an element in a balanced Binary Search Tree (BST)?",
						Points: 10,
						Options: []model.Option{
							{ID: 1, Text: "O(1)", IsCorrect: false},
							{ID: 2, Text: "O(log n)", IsCorrect: true},
							{ID: 3, Text: "O(n)", IsCorrect: false},
							{ID: 4, Text: "O(n log n)", IsCorrect: false},
						},
					},
					{
						ID:     2,
						Text:   "Which HTTP method is idempotent and typically used to replace or update a resource?",
						Points: 10,
						Options: []model.Option{
							{ID: 1, Text: "POST", IsCorrect: false},
							{ID: 2, Text: "PUT", IsCorrect: true},
							{ID: 3, Text: "PATCH", IsCorrect: false},
							{ID: 4, Text: "CONNECT", IsCorrect: false},
						},
					},
					{
						ID:     3,
						Text:   "In Go (Golang), how do you pass data concurrently and safely between goroutines without explicit lock mutexes?",
						Points: 10,
						Options: []model.Option{
							{ID: 1, Text: "Channels", IsCorrect: true},
							{ID: 2, Text: "Global Shared Memory", IsCorrect: false},
							{ID: 3, Text: "Thread Local Storage", IsCorrect: false},
							{ID: 4, Text: "Disk Files", IsCorrect: false},
						},
					},
					{
						ID:     4,
						Text:   "What is the primary benefit of an append-only architecture in high-concurrency systems?",
						Points: 10,
						Options: []model.Option{
							{ID: 1, Text: "Reduces table size automatically", IsCorrect: false},
							{ID: 2, Text: "Avoids row update lock contention and ensures immutable audit trails", IsCorrect: true},
							{ID: 3, Text: "Eliminates the need for any database indexes", IsCorrect: false},
							{ID: 4, Text: "Makes queries faster without RAM", IsCorrect: false},
						},
					},
					{
						ID:     5,
						Text:   "Which data structure follows the First-In, First-Out (FIFO) ordering principle?",
						Points: 10,
						Options: []model.Option{
							{ID: 1, Text: "Stack", IsCorrect: false},
							{ID: 2, Text: "Queue", IsCorrect: true},
							{ID: 3, Text: "Binary Heap", IsCorrect: false},
							{ID: 4, Text: "Hash Table", IsCorrect: false},
						},
					},
				},
			},
			{
				ID:              primitive.NewObjectID(),
				Title:           "General Engineering Assessment (All Levels)",
				Description:     "Universal college engineering & problem solving assessment with synchronized live timer.",
				LevelID:         0, // All levels
				GroupIDs:        []string{},
				DurationMinutes: 60,
				StartTime:       now.Add(-2 * time.Minute), // Started 2 mins ago
				EndTime:         now.Add(58 * time.Minute),
				IsActive:        true,
				CreatedAt:       now,
				Questions: []model.Question{
					{
						ID:     1,
						Text:   "Which principle states that the total energy in an isolated system remains constant?",
						Points: 15,
						Options: []model.Option{
							{ID: 1, Text: "Law of Conservation of Energy", IsCorrect: true},
							{ID: 2, Text: "Newton's Third Law", IsCorrect: false},
							{ID: 3, Text: "Bernoulli's Principle", IsCorrect: false},
							{ID: 4, Text: "Heisenberg Uncertainty Principle", IsCorrect: false},
						},
					},
					{
						ID:     2,
						Text:   "What does JWT stand for in web security and authentication?",
						Points: 15,
						Options: []model.Option{
							{ID: 1, Text: "Java Web Transport", IsCorrect: false},
							{ID: 2, Text: "JSON Web Token", IsCorrect: true},
							{ID: 3, Text: "Joint Worker Task", IsCorrect: false},
							{ID: 4, Text: "JavaScript Working Thread", IsCorrect: false},
						},
					},
				},
			},
		}

		for _, q := range sampleQuizzes {
			if err := s.quizRepo.Create(ctx, q); err != nil {
				log.Printf("Failed to seed sample quiz %s: %v", q.Title, err)
			}
		}
		log.Println("Sample quizzes created successfully.")
	}

	return nil
}
