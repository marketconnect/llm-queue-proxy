package main

import (
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/marketconnect/llm-queue-proxy/app/app"
)

func main() {
	a, err := app.NewApp()
	if err != nil {
		log.Printf("Application failed: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := a.Close(); err != nil {
			log.Printf("Error closing application: %v", err)
		}
	}()
	if err := a.Run(); err != nil {
		log.Printf("Application failed: %v", err)
		os.Exit(1)
	}
}
