package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

type Config struct {
	MongoDBURI     string
	JWTSecret      string
	Port           string
	Env            string
	BackendURL     string
	AllowedOrigins []string
	MongoClient    *mongo.Client
}

func LoadConfig() (*Config, error) {
	godotenv.Load()

	environment := getEnvOrDefault("ENV", "development")
	defaultOrigins := ""
	if environment == "development" {
		defaultOrigins = "http://localhost:5173,http://127.0.0.1:5173"
	}

	return &Config{
		MongoDBURI:     os.Getenv("MONGODB_URI"),
		JWTSecret:      getFirstEnv("JWT_SECRET", "JWT_ACCESS_SECRET"),
		Port:           getEnvOrDefault("PORT", "3001"),
		Env:            environment,
		BackendURL:     getEnvOrDefault("BACKEND_FLUT_URL", "http://localhost:8080/api/v1"),
		AllowedOrigins: splitCommaSeparated(getEnvOrDefault("ALLOWED_ORIGINS", defaultOrigins)),
	}, nil
}

func splitCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}

	values := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func getFirstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func getEnvOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func (c *Config) Validate() error {
	if c.MongoDBURI == "" {
		return fmt.Errorf("MONGODB_URI not set")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET not set")
	}
	return nil
}
