# Dependency Injection Refactoring

This document explains the dependency injection refactoring that was implemented to make the application more modular, testable, and maintainable.

## Changes Made

### 1. Storage → Repository Rename
- Renamed `app/internal/storage/` to `app/internal/repository/`
- Changed `Store` interface to `Repository` interface
- Updated all references from "storage" to "repository"
- Environment variable changed from `STORAGE_TYPE` to `REPOSITORY_TYPE`

### 2. Dependency Injection Pattern
- Removed singleton pattern from session manager
- All dependencies are now created and injected in `app.go`
- Configuration is injected instead of accessed globally

### 3. Handler Refactoring
- Converted function-based handlers to struct-based handlers
- Handlers now accept dependencies via constructor injection
- Example:
  ```go
  // Old pattern
  func ProxyHandler(w http.ResponseWriter, r *http.Request) {
      sessionManager := session.GetManager() // Global singleton
      // ...
  }
  
  // New pattern
  type ProxyHandler struct {
      sessionManager *session.SessionManager
  }
  
  func NewProxyHandler(sessionManager *session.SessionManager) *ProxyHandler {
      return &ProxyHandler{sessionManager: sessionManager}
  }
  
  func (ph *ProxyHandler) Handle(w http.ResponseWriter, r *http.Request) {
      // Use ph.sessionManager (injected dependency)
      // ...
  }
  ```

## Benefits

### 1. **Testability**
- Dependencies can be easily mocked for unit testing
- No global state to manage in tests
- Each component can be tested in isolation

### 2. **Modularity** 
- Clear separation of concerns
- Dependencies are explicit and visible
- Easier to understand component relationships

### 3. **Flexibility**
- Easy to swap implementations (memory vs SQLite repository)
- Configuration can be injected from different sources
- Supports different deployment scenarios

### 4. **Resource Management**
- Proper cleanup of resources (database connections, etc.)
- Clear lifecycle management
- No resource leaks from singleton patterns

## Usage Examples

### Basic Usage (Memory Repository)
```bash
go run ./app/cmd/main.go
# Uses in-memory repository by default
```

### SQLite Repository
```bash
REPOSITORY_TYPE=sqlite SQLITE_DSN=sessions.db go run ./app/cmd/main.go
# Note: Requires importing SQLite driver in main.go:
# import _ "github.com/mattn/go-sqlite3"
```

### Adding SQLite Support
To use SQLite, add to your `main.go`:
```go
import _ "github.com/mattn/go-sqlite3"
```

And to your `go.mod`:
```
go get github.com/mattn/go-sqlite3
```

## Architecture

```
app.go
├── Creates Dependencies struct
├── Initializes Repository (Memory or SQLite)
├── Creates SessionManager with Repository
├── Creates Handlers with SessionManager
└── Starts HTTP server

Dependencies:
┌─────────────┐    ┌──────────────────┐    ┌─────────────┐
│   Config    │    │   Repository     │    │ SessionMgr  │
│             │───▶│ (Memory/SQLite)  │───▶│             │
└─────────────┘    └──────────────────┘    └─────────────┘
                                                    │
                                                    ▼
                                           ┌─────────────┐
                                           │  Handlers   │
                                           │ (Proxy/Stat)│
                                           └─────────────┘
```

## Repository Interface

The `Repository` interface allows easy swapping of storage backends:

```go
type Repository interface {
    Init() error
    Close() error
    GetSession(sessionID string) (*SessionData, error)
    CreateSession(sessionID string) (*SessionData, error)
    UpdateSessionTokens(sessionID string, usage TokenUsage) (*SessionData, error)
    ListSessions() (map[string]*SessionData, error)
}
```

## Configuration

All configuration is now injected through the `Dependencies` struct:

```go
type Dependencies struct {
    Config         *config.Config
    Repository     repository.Repository
    SessionManager *session.SessionManager
}
```

This makes it easy to:
- Override configuration for testing
- Support multiple configuration sources
- Validate configuration at startup
- Provide sensible defaults

## Migration Guide

If you have existing code using the old patterns:

### Old Pattern:
```go
sessionManager := session.GetManager()
http.HandleFunc("/endpoint", handlers.ProxyHandler)
```

### New Pattern:
```go
deps, err := app.CreateDependencies()
if err != nil {
    log.Fatal(err)
}
defer deps.Close()

proxyHandler := handlers.NewProxyHandler(deps.SessionManager)
http.HandleFunc("/endpoint", proxyHandler.Handle)
``` 