package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/eythor/mcp-server/internal/database"
	"github.com/eythor/mcp-server/internal/debug"
	"github.com/eythor/mcp-server/internal/handlers"
	"github.com/eythor/mcp-server/internal/mcp"
)

type HTTPServer struct {
	mcpServer *mcp.Server
}

func NewHTTPServer(mcpServer *mcp.Server) *HTTPServer {
	return &HTTPServer{
		mcpServer: mcpServer,
	}
}

// Handle JSON-RPC requests over HTTP
func (h *HTTPServer) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	debug.Request(r.Method, r.URL.Path, nil)
	
	// Set CORS headers for browser access
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		debug.Response(http.StatusOK, "CORS preflight")
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		debug.Error("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		debug.Error("Invalid JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	debug.Verbose("HTTP request body: %s", string(request))

	response, err := h.mcpServer.HandleMessage(request)
	if err != nil {
		debug.Error("Error handling message: %v", err)
		log.Printf("Error handling message: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if response != nil {
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// Simple health check endpoint
func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "mcp-server",
	})
}

// Natural language query endpoint (REST-style)
func (h *HTTPServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var queryRequest struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&queryRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if queryRequest.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Create JSON-RPC request for natural language query
	rpcRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "natural_language_query",
			"arguments": map[string]interface{}{
				"query": queryRequest.Query,
			},
		},
		"id": 1,
	}

	requestBytes, _ := json.Marshal(rpcRequest)
	response, err := h.mcpServer.HandleMessage(requestBytes)
	if err != nil {
		log.Printf("Error handling query: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Extract the text from MCP response
	if response != nil && response.Result != nil {
		if resultMap, ok := response.Result.(map[string]interface{}); ok {
			if content, ok := resultMap["content"].([]map[string]interface{}); ok && len(content) > 0 {
				if text, ok := content[0]["text"].(string); ok {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]string{
						"response": text,
					})
					return
				}
			}
		}
	}

	http.Error(w, "Failed to process query", http.StatusInternalServerError)
}

func main() {
	debug.Log("HTTP server starting...")
	
	// Get configuration from environment
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./database.db"
	}
	debug.Log("Using database at: %s", dbPath)

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}
	debug.Verbose("OPENROUTER_API_KEY configured")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize database
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create MCP server
	handler := handlers.NewHandler(db, apiKey)
	mcpServer := mcp.NewServer(handler)

	// Create HTTP server
	httpServer := NewHTTPServer(mcpServer)

	// Set up routes
	http.HandleFunc("/", httpServer.handleHealth)
	http.HandleFunc("/health", httpServer.handleHealth)
	http.HandleFunc("/jsonrpc", httpServer.handleJSONRPC)
	http.HandleFunc("/query", httpServer.handleQuery)

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("MCP HTTP Server starting on %s", addr)
	log.Printf("Endpoints:")
	log.Printf("  POST /jsonrpc - JSON-RPC endpoint")
	log.Printf("  POST /query   - Natural language query endpoint")
	log.Printf("  GET  /health  - Health check")
	
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}