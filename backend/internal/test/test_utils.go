package internal

import (
	"fmt" // debug prints
	"testing"

	"github.com/bsmart/abis/internal/config"
	"github.com/bsmart/abis/internal/repository"
	"github.com/bsmart/abis/internal/service"
)

// Test configuration

// TestConfig returns a test configuration.
func TestConfig() config.Config {
	fmt.Println("[TEST] TestConfig")
	return config.Config{
		Port:         "8081",
		DatabasePath: "./data/test_abis.db",
		JWTSecret:    "test-secret-key",
	}
}

// Test utilities

// SetupTestDB initializes a test database.
func SetupTestDB(t *testing.T) (*repository.WorkflowRepo, func()) {
	fmt.Println("[TEST] SetupTestDB")
	// Implementation would go here
	return nil, nil
}
func CleanupTestDB(t *testing.T, db *repository.WorkflowRepo) {
	fmt.Println("[TEST] CleanupTestDB")
	// Implementation would go here
}

// Example test helper
func ExampleAuthService() *service.AuthService {
	fmt.Println("[TEST] ExampleAuthService")
	// Implementation would go here
	return nil
}
