package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"quiz-system/internal/config"
	"quiz-system/internal/presenter"
	"quiz-system/internal/repository"
	"quiz-system/internal/service"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting College Quiz System on port %s...", cfg.Port)

	// 1. Connect to MongoDB
	mongoDB, err := repository.NewMongoDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoDB.Client.Disconnect(ctx); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}()

	// 2. Parse HTML Templates with Custom Helpers
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"divFloat": func(a, b int) float64 {
			if b == 0 {
				return 0
			}
			return (float64(a) / float64(b)) * 100.0
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("invalid dict call: odd number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"seq": func(start, end int) []int {
			var s []int
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
		"min": func(a, b int) int {
			if a < b {
				return a
			}
			return b
		},
		"max": func(a, b int) int {
			if a > b {
				return a
			}
			return b
		},
		"formatTime": func(t time.Time, layout string) string {
			if t.IsZero() {
				return ""
			}
			return t.UTC().Format(layout)
		},
		"isoTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.UTC().Format(time.RFC3339)
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("web/templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// 3. Initialize Repositories
	studentRepo := repository.NewStudentRepository(mongoDB.Database)
	quizRepo := repository.NewQuizRepository(mongoDB.Database)
	answerRepo := repository.NewAnswerRepository(mongoDB.Database)
	sessionRepo := repository.NewSessionRepository(mongoDB.Database)
	resultRepo := repository.NewResultRepository(mongoDB.Database)

	// 4. Initialize Services
	authService := service.NewAuthService(studentRepo, cfg)
	quizService := service.NewQuizService(quizRepo, sessionRepo, answerRepo, resultRepo)
	answerService := service.NewAnswerService(answerRepo, quizRepo, sessionRepo, resultRepo)
	resultService := service.NewResultService(resultRepo, answerRepo, quizRepo, studentRepo)
	importService := service.NewImportService(studentRepo)
	seederService := service.NewSeederService(studentRepo, quizRepo)

	// 5. Seed Hierarchy & Quizzes
	ctxSeeder, cancelSeeder := context.WithTimeout(context.Background(), 60*time.Second)
	if err := seederService.SeedInitialData(ctxSeeder); err != nil {
		log.Printf("Warning: Seeder error: %v", err)
	}
	cancelSeeder()

	// 6. Initialize Presenters
	authPres := presenter.NewAuthPresenter(authService, tmpl)
	studentPres := presenter.NewStudentPresenter(quizService, resultService, tmpl)
	quizPres := presenter.NewQuizPresenter(quizService, answerService, resultService, tmpl)
	adminPres := presenter.NewAdminPresenter(quizService, resultService, studentRepo, importService, tmpl)

	// 7. Setup Router & Routes
	router := presenter.NewRouter(presenter.RouterDependencies{
		AuthService:    authService,
		AuthPresenter:  authPres,
		StudentPres:    studentPres,
		QuizPresenter:  quizPres,
		AdminPresenter: adminPres,
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("🚀 Server listening at http://localhost:%s\n", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-stopChan
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server gracefully stopped.")
}
