.PHONY: test test-and-commit help

# Default Go command
GO := go

# Go module directory (assuming go.mod is in 'app/')
GO_MODULE_DIR := app

# Test command
TEST_CMD := cd $(GO_MODULE_DIR) && $(GO) test -v ./...

# Default commit message
COMMIT_MESSAGE ?= "Automated commit: Tests passed"

test:
	@echo "Running tests in $(GO_MODULE_DIR)..."
	@$(TEST_CMD)

# Target to run tests and then Git commands if tests are OK
test-and-commit: test
	@echo "Tests passed. Proceeding with Git commands..."
	@git add .
	@git commit -m "$(COMMIT_MESSAGE)"
	@echo "Changes added and committed with message: $(COMMIT_MESSAGE)"
	@echo "Consider running 'git push' manually if needed."

# A simple help target
help:
	@echo "Available targets:"
	@echo "  test            - Run all Go tests within the '$(GO_MODULE_DIR)' directory."
	@echo "  test-and-commit - Run tests, then git add and commit if tests pass."
	@echo "                    Override commit message with: make test-and-commit COMMIT_MESSAGE=\\"Your message\\""
	@echo "  help            - Show this help message."

# Default target
default: help