.PHONY: build run dev test clean db-init http-server

build:
	go build -o mcp-server .

build-http:
	go build -o mcp-http-server cmd/http/main.go

run: build
	./mcp-server

run-http: build-http
	./mcp-http-server

dev:
	air

http-server:
	go run cmd/http/main.go

test:
	go test -v ./...

clean:
	rm -f mcp-server
	rm -rf tmp/
	# Let's comment this out for now
	# rm -f database.db

db-init:
	sqlite3 database.db < schema.sql

deps:
	go mod download
	go mod tidy

format:
	go fmt ./...
	
lint:
	golangci-lint run

install: build
	cp mcp-server /usr/local/bin/
