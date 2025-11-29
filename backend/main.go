package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	"github.com/eythor/mcp-server/internal/database"
	"github.com/eythor/mcp-server/internal/handlers"
	"github.com/eythor/mcp-server/internal/mcp"
)

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./database.db"
	}

	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}

	handler := handlers.NewHandler(db, apiKey)
	server := mcp.NewServer(handler)

	scanner := bufio.NewScanner(os.Stdin)
	writer := json.NewEncoder(os.Stdout)

	log.SetOutput(os.Stderr)
	log.Println("MCP Server started. Listening for JSON-RPC messages...")

	for scanner.Scan() {
		message := scanner.Bytes()
		
		response, err := server.HandleMessage(message)
		if err != nil {
			log.Printf("Error handling message: %v", err)
			continue
		}

		if response != nil {
			if err := writer.Encode(response); err != nil {
				log.Printf("Error encoding response: %v", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}
}
