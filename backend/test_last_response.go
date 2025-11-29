package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

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
	
	// Simulate setting a patient context
	fmt.Println("=== Setting patient context ===")
	handler.SetPatientContext("92213ec2-e8df-97db-b57b-b820adf52c6e") // Marty's ID
	
	// Manually set a last response to simulate a previous interaction
	fmt.Println("\n=== Simulating a previous response ===")
	handler.SetLastResponse("The patient's blood pressure reading of 135/85 indicates stage 1 hypertension. I recommend lifestyle modifications and regular monitoring.")
	
	// Get context info to see if last response is included
	contextInfo := handler.GetContextInfo()
	fmt.Printf("\n=== Context with last response ===\n%s\n", contextInfo)
	
	// Check if last response is included
	if strings.Contains(contextInfo, "Previous Response") {
		fmt.Println("\n✓ Last response is included in context")
	} else {
		fmt.Println("\n✗ Last response is NOT included in context")
	}
	
	// Now change patient - should clear last response
	fmt.Println("\n=== Changing patient (should clear last response) ===")
	handler.SetPatientContext("59cca175-3a5b-e3df-de3a-251f8a406635") // Different patient
	
	contextInfo = handler.GetContextInfo()
	if strings.Contains(contextInfo, "Previous Response") {
		fmt.Println("✗ Last response was NOT cleared when patient changed")
	} else {
		fmt.Println("✓ Last response was properly cleared when patient changed")
	}
	
	// Test with a long response that should be truncated
	fmt.Println("\n=== Testing with long response (should truncate) ===")
	longResponse := strings.Repeat("This is a very long response. ", 50)
	handler.SetLastResponse(longResponse)
	
	contextInfo = handler.GetContextInfo()
	if strings.Contains(contextInfo, "(truncated)") {
		fmt.Println("✓ Long response was properly truncated")
	} else {
		fmt.Println("✗ Long response was NOT truncated")
	}
}