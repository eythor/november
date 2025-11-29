package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/eythor/mcp-server/internal/handlers"
)

type Server struct {
	handler *handlers.Handler
}

func NewServer(handler *handlers.Handler) *Server {
	return &Server{
		handler: handler,
	}
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (s *Server) HandleMessage(message []byte) (*JSONRPCResponse, error) {
	var request JSONRPCRequest
	if err := json.Unmarshal(message, &request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	switch request.Method {
	case "initialize":
		response.Result = s.handleInitialize()
	case "initialized":
		// No response needed for initialized
		return nil, nil
	case "tools/list":
		response.Result = s.handleToolsList()
	case "tools/call":
		result, err := s.handleToolsCall(request.Params)
		if err != nil {
			response.Error = &Error{
				Code:    -32603,
				Message: err.Error(),
			}
		} else {
			response.Result = result
		}
	default:
		response.Error = &Error{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s", request.Method),
		}
	}

	return response, nil
}

func (s *Server) handleInitialize() map[string]interface{} {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "healthcare-mcp-server",
			"version": "0.1.0",
		},
	}
}

func (s *Server) handleToolsList() map[string]interface{} {
	tools := []map[string]interface{}{
		{
			"name":        "natural_language_query",
			"description": "Process natural language queries using AI with access to all healthcare tools",
			"inputSchema": map[string]interface{}{
				"type":     "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Natural language query about patients, medical history, appointments, etc.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "lookup_patient",
			"description": "Look up a patient by name or ID",
			"inputSchema": map[string]interface{}{
				"type":     "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Patient name or ID to search for",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "schedule_appointment",
			"description": "Schedule an appointment for a patient",
			"inputSchema": map[string]interface{}{
				"type":     "object",
				"properties": map[string]interface{}{
					"patient_id": map[string]interface{}{
						"type":        "string",
						"description": "Patient ID",
					},
					"practitioner_id": map[string]interface{}{
						"type":        "string",
						"description": "Practitioner ID",
					},
					"datetime": map[string]interface{}{
						"type":        "string",
						"description": "Appointment date and time (ISO 8601 format)",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Type of appointment",
					},
				},
				"required": []string{"patient_id", "practitioner_id", "datetime"},
			},
		},
		{
			"name":        "cancel_appointment",
			"description": "Cancel an appointment",
			"inputSchema": map[string]interface{}{
				"type":     "object",
				"properties": map[string]interface{}{
					"encounter_id": map[string]interface{}{
						"type":        "string",
						"description": "Encounter/Appointment ID to cancel",
					},
				},
				"required": []string{"encounter_id"},
			},
		},
		{
			"name":        "get_medical_history",
			"description": "Retrieve patient medical history",
			"inputSchema": map[string]interface{}{
				"type":     "object",
				"properties": map[string]interface{}{
					"patient_id": map[string]interface{}{
						"type":        "string",
						"description": "Patient ID",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category of history (conditions, medications, procedures, immunizations, allergies)",
						"enum":        []string{"conditions", "medications", "procedures", "immunizations", "allergies", "all"},
					},
				},
				"required": []string{"patient_id"},
			},
		},
		{
			"name":        "get_medication_info",
			"description": "Provide medication information",
			"inputSchema": map[string]interface{}{
				"type":     "object",
				"properties": map[string]interface{}{
					"medication_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the medication",
					},
				},
				"required": []string{"medication_name"},
			},
		},
		{
			"name":        "answer_health_question",
			"description": "Answer general health-related questions using AI",
			"inputSchema": map[string]interface{}{
				"type":     "object",
				"properties": map[string]interface{}{
					"question": map[string]interface{}{
						"type":        "string",
						"description": "Health-related question to answer",
					},
				},
				"required": []string{"question"},
			},
		},
	}

	return map[string]interface{}{
		"tools": tools,
	}
}

func (s *Server) handleToolsCall(params json.RawMessage) (interface{}, error) {
	var toolCall struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(params, &toolCall); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool call: %w", err)
	}

	switch toolCall.Name {
	case "natural_language_query":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.ProcessNaturalLanguageQuery(args.Query)

	case "lookup_patient":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.LookupPatient(args.Query)

	case "schedule_appointment":
		var args struct {
			PatientID       string `json:"patient_id"`
			PractitionerID  string `json:"practitioner_id"`
			DateTime        string `json:"datetime"`
			Type           string `json:"type"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.ScheduleAppointment(args.PatientID, args.PractitionerID, args.DateTime, args.Type)

	case "cancel_appointment":
		var args struct {
			EncounterID string `json:"encounter_id"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.CancelAppointment(args.EncounterID)

	case "get_medical_history":
		var args struct {
			PatientID string `json:"patient_id"`
			Category  string `json:"category"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		if args.Category == "" {
			args.Category = "all"
		}
		return s.handler.GetMedicalHistory(args.PatientID, args.Category)

	case "get_medication_info":
		var args struct {
			MedicationName string `json:"medication_name"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.GetMedicationInfo(args.MedicationName)

	case "answer_health_question":
		var args struct {
			Question string `json:"question"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.AnswerHealthQuestion(args.Question)

	default:
		return nil, fmt.Errorf("unknown tool: %s", toolCall.Name)
	}
}
