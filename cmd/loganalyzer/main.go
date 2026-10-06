package main

import (
	"log"

	"LogAnalyzer/internal/app"
)

// main starts the LogAnalyzer application.
func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
