.PHONY: build run dev test clean db-init

build:
	go build -o mcp-server .

run: build
	./mcp-server

dev:
	air

test:
	go test -v ./...

clean:
	rm -f mcp-server
	rm -rf tmp/
	rm -f database.db

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
