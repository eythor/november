package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	"github.com/eythor/mcp-server/internal/database"
	"github.com/eythor/mcp-server/internal/debug"
	"github.com/eythor/mcp-server/internal/handlers"
	"github.com/eythor/mcp-server/internal/mcp"
	"github.com/joho/godotenv"
)

func main() {
	debug.Log("MCP server starting...")
	
	// Load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()
	debug.Verbose("Environment loaded")

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./database.db"
	}
	debug.Log("Using database at: %s", dbPath)

	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	debug.Verbose("Database initialized")

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}
	debug.Verbose("OPENROUTER_API_KEY configured")

	handler := handlers.NewHandler(db, apiKey)
	server := mcp.NewServer(handler)
	debug.Verbose("MCP server initialized")

	scanner := bufio.NewScanner(os.Stdin)
	writer := json.NewEncoder(os.Stdout)

	log.SetOutput(os.Stderr)
	log.Println("MCP Server started. Listening for JSON-RPC messages...")
	debug.Log("MCP server ready, debug mode: %s", os.Getenv("MCP_DEBUG"))

	for scanner.Scan() {
		message := scanner.Bytes()
		debug.Trace("Received message: %s", string(message))
		
		response, err := server.HandleMessage(message)
		if err != nil {
			debug.Error("Error handling message: %v", err)
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
