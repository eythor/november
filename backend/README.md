# Healthcare MCP Server

This is an MCP (Model Context Protocol) server implementation in Go that provides healthcare-related tools. The server integrates with FHIR healthcare data stored in SQLite and uses OpenRouter API for AI-powered responses.

## Features

The MCP server provides the following tools:

- **lookup_patient** - Look up patients by name or ID
- **schedule_appointment** - Schedule appointments for patients
- **cancel_appointment** - Cancel existing appointments
- **get_medical_history** - Retrieve patient medical history (conditions, medications, procedures, immunizations, allergies)
- **get_medication_info** - Get information about medications using AI
- **answer_health_question** - Answer general health-related questions using AI

## Prerequisites

- Nix with flakes enabled
- OpenRouter API key

## Development Setup

1. Enter the development environment:
```bash
nix develop
```

2. Set your OpenRouter API key:
```bash
export OPENROUTER_API_KEY="your-api-key-here"
```

3. Initialize the database (if not already created):
```bash
make db-init
```

## Usage

### Build and Run
```bash
# Build the binary
make build

# Run the server
make run

# Or run with hot reload during development
make dev
```

### Running in MCP Mode
The server communicates via JSON-RPC over stdin/stdout. Each message should be a JSON-RPC 2.0 formatted request.

Example initialization:
```json
{"jsonrpc": "2.0", "method": "initialize", "params": {}, "id": 1}
```

### Environment Variables

- `OPENROUTER_API_KEY` - Required. Your OpenRouter API key
- `DATABASE_PATH` - Optional. Path to SQLite database (default: ./database.db)

## Database Schema

The server uses a comprehensive FHIR-based healthcare database schema with tables for:
- Patients, Practitioners, Organizations, Locations
- Encounters, Conditions, Observations, Procedures
- Medications, Immunizations, Allergies
- Claims, Care Plans, and more

See `schema.sql` for the complete schema definition.

## Development Commands

```bash
make build      # Build the binary
make run        # Build and run
make dev        # Run with hot reload (air)
make test       # Run tests
make clean      # Clean build artifacts
make db-init    # Initialize database
make format     # Format Go code
make lint       # Run linter
```

## Project Structure

```
.
├── main.go                    # Entry point
├── internal/
│   ├── mcp/
│   │   └── server.go         # MCP protocol implementation
│   ├── handlers/
│   │   └── handler.go        # Tool handlers with OpenRouter integration
│   └── database/
│       └── db.go             # Database connection and models
├── schema.sql                # Database schema
├── flake.nix                 # Nix development environment
├── go.mod                    # Go module definition
├── Makefile                  # Build automation
└── README.md                 # This file
```

## Contributing

1. Make sure to run `make format` and `make lint` before committing
2. Add tests for new functionality
3. Update documentation as needed