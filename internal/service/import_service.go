package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"quiz-system/internal/model"
	"quiz-system/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportService interface {
	ParseStudentsFromCSV(reader io.Reader) ([]model.Student, error)
	BulkImportStudents(ctx context.Context, students []model.Student) (int, error)
	ImportStudentsFromCSV(ctx context.Context, reader io.Reader) (int, error)
	GenerateSampleCSV() []byte
}

type importService struct {
	studentRepo repository.StudentRepository
}

func NewImportService(studentRepo repository.StudentRepository) ImportService {
	return &importService{
		studentRepo: studentRepo,
	}
}

func (s *importService) ParseStudentsFromCSV(reader io.Reader) ([]model.Student, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV format: %w", err)
	}

	if len(records) == 0 {
		return nil, errors.New("CSV file is empty")
	}

	// Detect header row
	startIndex := 0
	codeCol, nameCol, levelCol, groupCol := 0, 1, 2, 3
	firstRow := records[0]

	if isHeaderRow(firstRow) {
		startIndex = 1
		for idx, col := range firstRow {
			clean := strings.ToLower(strings.TrimSpace(col))
			switch {
			case strings.Contains(clean, "code") || strings.Contains(clean, "id") || strings.Contains(clean, "student_id"):
				codeCol = idx
			case strings.Contains(clean, "name") || strings.Contains(clean, "student_name"):
				nameCol = idx
			case strings.Contains(clean, "level"):
				levelCol = idx
			case strings.Contains(clean, "group") || strings.Contains(clean, "section"):
				groupCol = idx
			}
		}
	}

	var students []model.Student
	now := time.Now().UTC()

	for rowIdx := startIndex; rowIdx < len(records); rowIdx++ {
		row := records[rowIdx]
		if len(row) < 2 {
			continue
		}

		code := ""
		if codeCol < len(row) {
			code = strings.ToUpper(strings.TrimSpace(row[codeCol]))
		}
		name := ""
		if nameCol < len(row) {
			name = strings.TrimSpace(row[nameCol])
		}
		levelStr := ""
		if levelCol < len(row) {
			levelStr = strings.TrimSpace(row[levelCol])
		}
		group := ""
		if groupCol < len(row) {
			group = strings.ToUpper(strings.TrimSpace(row[groupCol]))
		}

		if code == "" || name == "" {
			continue
		}

		level, err := strconv.Atoi(levelStr)
		if err != nil || level < 1 || level > 5 {
			level = 1
		}

		if group == "" {
			group = "A"
		}

		students = append(students, model.Student{
			ID:          primitive.NewObjectID(),
			StudentCode: code,
			Name:        name,
			LevelID:     level,
			GroupID:     group,
			IsActive:    true,
			CreatedAt:   now,
		})
	}

	if len(students) == 0 {
		return nil, errors.New("no valid student records found in CSV file")
	}

	return students, nil
}

func (s *importService) BulkImportStudents(ctx context.Context, students []model.Student) (int, error) {
	if len(students) == 0 {
		return 0, nil
	}
	return s.studentRepo.BulkUpsert(ctx, students)
}

func (s *importService) ImportStudentsFromCSV(ctx context.Context, reader io.Reader) (int, error) {
	students, err := s.ParseStudentsFromCSV(reader)
	if err != nil {
		return 0, err
	}
	return s.BulkImportStudents(ctx, students)
}

func isHeaderRow(row []string) bool {
	if len(row) == 0 {
		return false
	}
	combined := strings.ToLower(strings.Join(row, " "))
	return strings.Contains(combined, "student") ||
		strings.Contains(combined, "code") ||
		strings.Contains(combined, "name") ||
		strings.Contains(combined, "level") ||
		strings.Contains(combined, "group")
}

func (s *importService) GenerateSampleCSV() []byte {
	var sb strings.Builder
	sb.WriteString("student_code,name,level,group\n")
	sb.WriteString("L1A-001,Ahmed Ali,1,A\n")
	sb.WriteString("L1A-002,Fatima Zahra,1,A\n")
	sb.WriteString("L1B-001,Omar Hassan,1,B\n")
	sb.WriteString("L2A-001,Sarah Mansour,2,A\n")
	sb.WriteString("L3C-001,Khaled Ibrahim,3,C\n")
	sb.WriteString("L4D-001,Mariam Youssef,4,D\n")
	return []byte(sb.String())
}
