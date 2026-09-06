package internal

import (
	"testing"

	"github.com/bsmart/abis/internal/chat"
	"github.com/bsmart/abis/internal/config"
	"github.com/bsmart/abis/internal/database"
	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/handler"
	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/middleware"
	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/repository"
	"github.com/bsmart/abis/internal/service"
)

// Test configuration

// TestConfig returns a test configuration.
func TestConfig() config.Config {
	return config.Config{
		Port:         "8081",
		DatabasePath: "./data/test_abis.db",
		JWTSecret:    "test-secret-key",
		GroqAPIKey:   "test-groq-key",
		GroqModel:    "test-model",
	}
}

// Test utilities

// SetupTestDB initializes a test database.
func SetupTestDB(t *testing.T) (*database.DB, func())
func CleanupTestDB(t *testing.T, db *database.DB)
