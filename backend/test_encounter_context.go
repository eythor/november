package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/eythor/mcp-server/internal/handlers"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Open database
	dbPath := "./database.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create handler
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		apiKey = "test-key"
	}
	
	handler := handlers.NewHandler(db, apiKey)
	
	// Look up a patient to set context
	fmt.Println("Looking up patient 'Marty'...")
	_, err = handler.LookupPatient("Marty")
	if err != nil {
		log.Fatalf("Failed to lookup patient: %v", err)
	}
	
	// Get the context info that would be sent to LLM
	contextInfo := handler.GetContextInfo()
	fmt.Printf("\n=== Context info for LLM ===\n%s\n", contextInfo)
	
	// Also test with a patient that might have more encounters
	fmt.Println("\n\nLooking up patient 'Cole'...")
	_, err = handler.LookupPatient("Cole")
	if err != nil {
		fmt.Printf("Failed to lookup Cole: %v\n", err)
	} else {
		contextInfo = handler.GetContextInfo()
		fmt.Printf("\n=== Context after searching Cole ===\n%s\n", contextInfo)
	}
}