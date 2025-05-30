package app_test

import (
	"os"
	"testing"

	"github.com/marketconnect/llm-queue-proxy/app/app"
	"github.com/marketconnect/llm-queue-proxy/app/internal/repository"
)

func TestNewApp_DefaultConfig(t *testing.T) {
	// Ensure critical env vars are set for default run
	os.Setenv("OPENAI_API_KEY", "test_api_key")
	// Default is memory repository, so DSN is not strictly needed unless type is sqlite
	os.Setenv("REPOSITORY_TYPE", "memory")

	a, err := app.NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	if a == nil {
		t.Fatal("NewApp() returned nil app")
	}
	defer a.Close()

	if a.Config == nil {
		t.Error("App.Config is nil")
	}
	if a.Repository == nil {
		t.Error("App.Repository is nil")
	}
	if _, ok := a.Repository.(*repository.MemoryRepository); !ok {
		t.Errorf("Expected Repository to be *MemoryRepository, got %T", a.Repository)
	}
	if a.SessionManager == nil {
		t.Error("App.SessionManager is nil")
	}
	if a.Queue == nil {
		t.Error("App.Queue is nil")
	}
}

func TestApp_Close(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test_api_key_close")
	os.Setenv("REPOSITORY_TYPE", "memory")

	a, err := app.NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	err = a.Close()
	if err != nil {
		t.Errorf("App.Close() failed: %v", err)
	}

	// Test double close
	err = a.Close()
	if err != nil {
		t.Errorf("App.Close() on already closed app failed: %v", err)
	}
}

// Note: Testing NewApp with SQLite repository type is tricky due to config singleton.
// It's better to test SQLiteRepository independently.
// Run() is not unit tested here as it starts an HTTP server. Integration tests would cover it.
