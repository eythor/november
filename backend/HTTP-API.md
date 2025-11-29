# HTTP API Documentation

The HTTP server provides RESTful endpoints for healthcare data management.

## Running the Server

```bash
# Using make
make http-server

# Or build and run
make build-http
./mcp-http-server

# With custom port
PORT=3000 ./mcp-http-server
```

## Base URL
Default: `http://localhost:8080`

## Endpoints

### Health Check
```
GET /health
```
Returns server health status.

### Patient Operations

#### Lookup Patient
```
GET /api/patient/lookup?q=<query>
```
Search for patients by name or ID. **Automatically sets context when single patient found.**

Example:
```bash
curl "http://localhost:8080/api/patient/lookup?q=John%20Doe"
```

#### Get Medical History
```
GET /api/patient/history?patient_id=<id>&category=<category>
```
- `patient_id` (optional): Uses context if not provided
- `category` (optional): Filter by category (medications, conditions, etc.)

### Appointment Operations

#### Schedule Appointment
```
POST /api/appointment/schedule
```
Body:
```json
{
  "patient_id": "123",        // Optional if context set
  "practitioner_id": "456",   // Optional if context set
  "datetime": "2024-01-15T14:00:00Z",
  "type": "General Consultation"
}
```

#### Cancel Appointment
```
POST /api/appointment/cancel?encounter_id=<id>
```

### Context Management

#### Set Patient Context
```
POST /api/context/patient
```
Body:
```json
{
  "patient_id": "123"
}
```

#### Set Practitioner Context
```
POST /api/context/practitioner
```
Body:
```json
{
  "practitioner_id": "456"
}
```

#### Get Current Context
```
GET /api/context
```

#### Clear Context
```
POST /api/context/clear
```

### Natural Language Query
```
POST /api/query
```
Body:
```json
{
  "query": "What medications is John Doe taking?"
}
```

## Response Format

Success:
```json
{
  "result": "Response text or data"
}
```

Error:
```json
{
  "error": "Error message"
}
```

## Context Auto-Setting

When you lookup a patient and only one match is found, the system automatically sets that patient as the default context. This means subsequent operations that require a patient ID will use this context if no explicit patient ID is provided.

Example workflow:
1. `GET /api/patient/lookup?q=John%20Doe` - Finds John Doe and sets as context
2. `GET /api/patient/history` - Automatically uses John Doe's ID
3. `POST /api/appointment/schedule` - Can omit patient_id, uses John Doe

## CORS

All endpoints support CORS for cross-origin requests from web applications.