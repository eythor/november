.PHONY: build run dev test clean db-init http-server start db-check

DATABASE_PATH ?= ./database.db

build:
	go build -o mcp-server .

build-http:
	go build -o mcp-http-server ./cmd/http-server

run: build
	./mcp-server

run-http: build-http db-check
	./mcp-http-server

dev:
	air

# Check if database exists, initialize if not
db-check:
	@if [ ! -f $(DATABASE_PATH) ]; then \
		echo "Database not found at $(DATABASE_PATH), initializing..."; \
		$(MAKE) db-init; \
	fi

# Start HTTP server (ensures database exists)
start: db-check
	@echo "Starting HTTP server on port 8080..."
	@echo "Database: $(DATABASE_PATH)"
	@echo "Make sure OPENROUTER_API_KEY is set"
	DATABASE_PATH=$(DATABASE_PATH) go run cmd/http/main.go

# Start HTTP server (alias for start)
http-server: start

test:
	go test -v ./...

clean:
	rm -f mcp-server
	rm -rf tmp/
	# Let's comment this out for now
	# rm -f database.db

db-init:
	@echo "Initializing database at $(DATABASE_PATH)..."
	sqlite3 $(DATABASE_PATH) < schema.sql
	@echo "Database initialized successfully"

deps:
	go mod download
	go mod tidy

format:
	go fmt ./...
	
lint:
	golangci-lint run

install: build
	cp mcp-server /usr/local/bin/
