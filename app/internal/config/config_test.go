package config_test

import (
	"os"
	"testing"

	"github.com/marketconnect/llm-queue-proxy/app/internal/config"
)

func TestGetConfig_Singleton(t *testing.T) {
	// Set a dummy API key for the test if not set, to avoid fatal error
	if os.Getenv("OPENAI_API_KEY") == "" {
		os.Setenv("OPENAI_API_KEY", "test_dummy_key_singleton")
		defer os.Unsetenv("OPENAI_API_KEY")
	}

	cfg1 := config.GetConfig()
	if cfg1 == nil {
		t.Fatal("GetConfig() returned nil on first call")
	}

	cfg2 := config.GetConfig()
	if cfg2 == nil {
		t.Fatal("GetConfig() returned nil on second call")
	}

	if cfg1 != cfg2 {
		t.Error("GetConfig() returned different instances, expected singleton behavior")
	}

	// Basic check for default values (assuming no conflicting env vars are set for these)
	// This depends on the environment the test is run in.
	// A more robust test would involve clearing relevant env vars.
}
