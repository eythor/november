# Healthcare MCP Server

This is an MCP (Model Context Protocol) server implementation in Go that provides healthcare-related tools. The server integrates with FHIR healthcare data stored in SQLite and uses OpenRouter API for AI-powered responses.

## Features

The MCP server provides the following tools:

- **lookup_patient** - Look up patients by name or ID (automatically sets context when single patient found)
- **schedule_appointment** - Schedule appointments for patients
- **cancel_appointment** - Cancel existing appointments
- **get_medical_history** - Retrieve patient medical history (conditions, medications, procedures, immunizations, allergies, observations)
- **get_medication_info** - Get information about medications using AI
- **get_medical_guidelines** - Get comprehensive medical guidelines, dosages, treatment protocols, and clinical best practices using AI
- **answer_health_question** - Answer general health-related questions using AI

### Context Management Tools:
- **set_patient_context** - Set default patient for subsequent operations
- **set_practitioner_context** - Set default practitioner for subsequent operations
- **get_context** - View current context settings
- **clear_context** - Clear all context settings

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

### Quick Natural Language Queries
```bash
# Enter nix development environment
nix develop

# Interactive mode - keep asking questions
./mcp-query

# Single query mode
./mcp-query "Find patient Cole117 and tell me about them"
./mcp-query "What medications is patient 123 taking?"
```

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

### Natural Language Query Examples
```json
{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "natural_language_query", "arguments": {"query": "Find all patients named John"}}, "id": 1}
{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "natural_language_query", "arguments": {"query": "Get medical history for patient ID abc123"}}, "id": 2}
```

### Medical Guidelines Examples
```json
{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "get_medical_guidelines", "arguments": {"query": "What is the recommended dosing for metformin in type 2 diabetes?"}}, "id": 1}
{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "get_medical_guidelines", "arguments": {"query": "Current hypertension treatment guidelines for adults"}}, "id": 2}
{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "get_medical_guidelines", "arguments": {"query": "Warfarin INR monitoring protocol"}}, "id": 3}
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
