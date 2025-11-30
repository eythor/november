package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/eythor/mcp-server/internal/debug"
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
	debug.Trace("MCP HandleMessage received: %s", string(message))

	var request JSONRPCRequest
	if err := json.Unmarshal(message, &request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	debug.Log("MCP handling method: %s", request.Method)

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
		debug.Verbose("Processing tools/call with params: %s", string(request.Params))
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
				"type": "object",
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
			"name":        "set_patient_context",
			"description": "Set the current patient ID for subsequent operations",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"patient_id": map[string]interface{}{
						"type":        "string",
						"description": "Patient ID to set as default",
					},
				},
				"required": []string{"patient_id"},
			},
		},
		{
			"name":        "set_practitioner_context",
			"description": "Set the current practitioner ID for subsequent operations",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"practitioner_id": map[string]interface{}{
						"type":        "string",
						"description": "Practitioner ID to set as default",
					},
				},
				"required": []string{"practitioner_id"},
			},
		},
		{
			"name":        "get_context",
			"description": "Get the current context (current patient and practitioner)",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "clear_context",
			"description": "Clear the current context (remove current patient and practitioner)",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "lookup_patient",
			"description": "Look up a patient by name or ID. Returns patient information including: name, patient ID, gender, birth_date (date of birth), age (calculated automatically), phone number, and location. Birth date and age are always included when available in the patient record.",
			"inputSchema": map[string]interface{}{
				"type": "object",
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
				"type": "object",
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
						"description": "Appointment date and time. Supports: ISO 8601 (2024-12-01T14:00:00+01:00), simple formats (2024-12-01 14:00), German format (01.12.2024 14:30), or natural language (tomorrow at 2pm, next Monday, in 3 days). System handles timezone conversion to Berlin automatically.",
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
				"type": "object",
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
				"type": "object",
				"properties": map[string]interface{}{
					"patient_id": map[string]interface{}{
						"type":        "string",
						"description": "Patient ID",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category of history (conditions, medications, procedures, immunizations, allergies, observations)",
						"enum":        []string{"conditions", "medications", "procedures", "immunizations", "allergies", "observations", "all"},
					},
				},
				"required": []string{"patient_id"},
			},
		},
		{
			"name":        "get_medication_info",
			"description": "Provide medication information",
			"inputSchema": map[string]interface{}{
				"type": "object",
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
			"name":        "get_medical_guidelines",
			"description": "Get comprehensive medical guidelines, medication dosages, treatment protocols, and clinical best practices",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Medical query (e.g., 'metformin dosing for type 2 diabetes', 'hypertension treatment guidelines', 'warfarin INR monitoring')",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "answer_health_question",
			"description": "Answer general health-related questions using AI",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"question": map[string]interface{}{
						"type":        "string",
						"description": "Health-related question to answer",
					},
				},
				"required": []string{"question"},
			},
		},
		{
			"name":        "add_observation",
			"description": "Add an observation record for a patient (e.g., vital signs, lab results, measurements)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"patient_id": map[string]interface{}{
						"type":        "string",
						"description": "Patient ID (optional if patient context is set)",
					},
					"code": map[string]interface{}{
						"type":        "string",
						"description": "Observation code (e.g., LOINC code)",
					},
					"display": map[string]interface{}{
						"type":        "string",
						"description": "Human-readable name of the observation (e.g., 'Body Weight', 'Blood Pressure')",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category of observation (e.g., 'vital-signs', 'laboratory', 'exam')",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Status of the observation (default: 'final')",
					},
					"effective_datetime": map[string]interface{}{
						"type":        "string",
						"description": "Date and time when observation was made (ISO 8601 format, defaults to now)",
					},
					"value_quantity": map[string]interface{}{
						"type":        "number",
						"description": "Numeric value of the observation (if applicable)",
					},
					"value_unit": map[string]interface{}{
						"type":        "string",
						"description": "Unit of measurement (e.g., 'kg', 'mmHg', 'mg/dL')",
					},
					"value_string": map[string]interface{}{
						"type":        "string",
						"description": "String value of the observation (if not numeric)",
					},
				},
				"required": []string{"code", "display"},
			},
		},
		{
			"name":        "calculate_age",
			"description": "Calculate the age of a patient from their birth date. Returns the patient's current age in years based on their birth date stored in the database. Uses patient context if patient_id is not provided.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"patient_id": map[string]interface{}{
						"type":        "string",
						"description": "Patient ID (optional if patient context is set)",
					},
				},
				"required": []string{},
			},
		},
		{
			"name":        "update_patient_birth_date",
			"description": "Update a patient's birth date in the database. Uses patient context if patient_id is not provided.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"patient_id": map[string]interface{}{
						"type":        "string",
						"description": "Patient ID (optional if patient context is set)",
					},
					"birth_date": map[string]interface{}{
						"type":        "string",
						"description": "Birth date in YYYY-MM-DD format (ISO 8601)",
					},
				},
				"required": []string{"birth_date"},
			},
		},
		{
			"name":        "confirm_date_choice",
			"description": "Confirm a date interpretation choice when an ambiguous date was provided. Use this when the user responds with A, B, or another choice letter to a date confirmation question.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"choice": map[string]interface{}{
						"type":        "string",
						"description": "The choice key (A, B, etc.) selected by the user",
					},
				},
				"required": []string{"choice"},
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

	debug.Log("MCP tool call: %s", toolCall.Name)
	debug.Verbose("Tool arguments: %s", string(toolCall.Arguments))

	switch toolCall.Name {
	case "natural_language_query":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.ProcessNaturalLanguageQuery(args.Query, "")

	case "set_patient_context":
		var args struct {
			PatientID string `json:"patient_id"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.SetPatientContext(args.PatientID)

	case "set_practitioner_context":
		var args struct {
			PractitionerID string `json:"practitioner_id"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.SetPractitionerContext(args.PractitionerID)

	case "get_context":
		return s.handler.GetContext()

	case "clear_context":
		return s.handler.ClearContext()

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
			PatientID      string `json:"patient_id"`
			PractitionerID string `json:"practitioner_id"`
			DateTime       string `json:"datetime"`
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

	case "get_medical_guidelines":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.GetMedicalGuidelines(args.Query)

	case "answer_health_question":
		var args struct {
			Question string `json:"question"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.AnswerHealthQuestion(args.Question)

	case "add_observation":
		var args struct {
			PatientID         string   `json:"patient_id"`
			Code              string   `json:"code"`
			Display           string   `json:"display"`
			Category          string   `json:"category"`
			Status            string   `json:"status"`
			EffectiveDateTime string   `json:"effective_datetime"`
			ValueQuantity     *float64 `json:"value_quantity"`
			ValueUnit         *string  `json:"value_unit"`
			ValueString       *string  `json:"value_string"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.AddObservation(args.PatientID, args.Code, args.Display, args.Category, args.Status, args.EffectiveDateTime, args.ValueQuantity, args.ValueUnit, args.ValueString)

	case "calculate_age":
		var args struct {
			PatientID string `json:"patient_id"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.CalculateAge(args.PatientID)

	case "update_patient_birth_date":
		var args struct {
			PatientID string `json:"patient_id"`
			BirthDate string `json:"birth_date"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.UpdatePatientBirthDate(args.PatientID, args.BirthDate)

	case "confirm_date_choice":
		var args struct {
			Choice string `json:"choice"`
		}
		if err := json.Unmarshal(toolCall.Arguments, &args); err != nil {
			return nil, err
		}
		return s.handler.ConfirmDateChoice(args.Choice)

	default:
		return nil, fmt.Errorf("unknown tool: %s", toolCall.Name)
	}
}
