package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port             string
	MongoURI         string
	DBName           string
	JWTSecret        []byte
	JWTExpiry        time.Duration
	AdminPIN         string
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	dbName := getEnv("DB_NAME", "quiz_system")
	jwtSecret := getEnv("JWT_SECRET", "college-quiz-jwt-secret-key-2026")
	adminPIN := getEnv("ADMIN_PIN", "admin123")

	expiryMinutesStr := getEnv("JWT_EXPIRY_MINUTES", "45")
	expiryMinutes, err := strconv.Atoi(expiryMinutesStr)
	if err != nil || expiryMinutes <= 0 {
		expiryMinutes = 45
	}

	return &Config{
		Port:             port,
		MongoURI:         mongoURI,
		DBName:           dbName,
		JWTSecret:        []byte(jwtSecret),
		JWTExpiry:        time.Duration(expiryMinutes) * time.Minute,
		AdminPIN:         adminPIN,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
