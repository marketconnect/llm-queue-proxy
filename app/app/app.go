package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/marketconnect/llm-queue-proxy/app/internal/config"
	"github.com/marketconnect/llm-queue-proxy/app/internal/handlers"
	"github.com/marketconnect/llm-queue-proxy/app/internal/queue"
	"github.com/marketconnect/llm-queue-proxy/app/internal/repository"
	"github.com/marketconnect/llm-queue-proxy/app/internal/session"
)

// App holds all application dependencies
type App struct {
	Config         *config.Config
	Repository     repository.Repository
	SessionManager *session.SessionManager
	Queue          *queue.Queue
}

// NewApp creates and initializes all application dependencies
func NewApp() (*App, error) {
	// Load configuration
	cfg := config.GetConfig()

	// Create repository based on configuration
	var repo repository.Repository
	var err error

	log.Printf("Initializing session repository with type: %s", cfg.Repository.Type)

	switch cfg.Repository.Type {
	case "sqlite":
		repo, err = repository.NewSQLiteRepository(cfg.Repository.SQLiteDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SQLite repository: %w", err)
		}
	case "memory":
		fallthrough
	default:
		repo = repository.NewMemoryRepository()
	}

	// Initialize repository
	if err := repo.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Create session manager with repository dependency
	sessionManager := session.NewSessionManager(repo)

	// Create queue with config dependency
	queueInstance := queue.NewQueue(cfg.OpenAI.RateLimitPerMin, cfg.OpenAI.BaseURL, cfg.OpenAI.APIKey)

	return &App{
		Config:         cfg,
		Repository:     repo,
		SessionManager: sessionManager,
		Queue:          queueInstance,
	}, nil
}

// Close cleans up all dependencies
func (a *App) Close() error {
	if a.Queue != nil {
		a.Queue.Close()
	}
	if a.SessionManager != nil {
		if err := a.SessionManager.Close(); err != nil {
			return fmt.Errorf("failed to close session manager: %w", err)
		}
	}
	return nil
}

// Run starts the HTTP server and registers handlers.
// The App instance `a` should be fully initialized before calling Run.
func (a *App) Run() error {
	// Create handler with injected dependencies
	proxyHandler := handlers.NewProxyHandler(a.SessionManager, a.Queue)
	sessionStatusHandler := handlers.NewSessionStatusHandler(a.SessionManager)

	// Setup routes
	http.HandleFunc("/v1/session/", proxyHandler.Handle)
	http.HandleFunc("/sessions/status", sessionStatusHandler.HandleSingle)

	addr := fmt.Sprintf(":%d", a.Config.HTTP.Port)
	log.Printf("Starting server on %s", addr)
	log.Printf("Available endpoints:")
	log.Printf("  - Proxy (session): /v1/session/{sessionID}/...")
	log.Printf("  - Session stats: /sessions/status")
	return http.ListenAndServe(addr, nil)
}
