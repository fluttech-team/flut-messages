package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

type Config struct {
	MongoDBURI    string
	JWTSecret     string
	Port          string
	Env           string
	BackendURL    string
	MongoClient   *mongo.Client
}

func LoadConfig() (*Config, error) {
	godotenv.Load()

	return &Config{
		MongoDBURI: os.Getenv("MONGODB_URI"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
		Port:       getEnvOrDefault("PORT", "3001"),
		Env:        getEnvOrDefault("ENV", "development"),
		BackendURL: getEnvOrDefault("BACKEND_FLUT_URL", "http://localhost:8080/api/v1"),
	}, nil
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
