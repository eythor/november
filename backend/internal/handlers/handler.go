package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eythor/mcp-server/internal/database"
	"github.com/google/uuid"
)

type Context struct {
	PatientID      string `json:"patient_id,omitempty"`
	PractitionerID string `json:"practitioner_id,omitempty"`
}

type Handler struct {
	db      *sql.DB
	apiKey  string
	context Context
	mu      sync.RWMutex
}

func NewHandler(db *sql.DB, apiKey string) *Handler {
	return &Handler{
		db:     db,
		apiKey: apiKey,
	}
}

func (h *Handler) LookupPatient(query string) (interface{}, error) {
	query = strings.TrimSpace(query)
	
	// Try exact ID lookup first
	patient, err := database.GetPatientByID(h.db, query)
	if err == nil {
		// Auto-set context for single patient found
		h.mu.Lock()
		h.context.PatientID = patient.ID
		h.mu.Unlock()
		
		resultText := formatPatientInfo(*patient)
		resultText += fmt.Sprintf("\n\n✓ Context updated: Default patient set to %s %s (ID: %s)", 
			patient.GivenName, patient.FamilyName, patient.ID)
		
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": resultText,
				},
			},
		}, nil
	}
	
	// Try name search
	patients, err := database.SearchPatientsByName(h.db, query)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	
	if len(patients) == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("No patients found matching '%s'", query),
				},
			},
		}, nil
	}
	
	// If exactly one patient found, auto-set context
	if len(patients) == 1 {
		p := patients[0]
		h.mu.Lock()
		h.context.PatientID = p.ID
		h.mu.Unlock()
		
		resultText := formatPatientInfo(p)
		resultText += fmt.Sprintf("\n\n✓ Context updated: Default patient set to %s %s (ID: %s)", 
			p.GivenName, p.FamilyName, p.ID)
		
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": resultText,
				},
			},
		}, nil
	}
	
	// Multiple patients found - don't auto-set context
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d patients matching '%s':\n\n", len(patients), query))
	for _, p := range patients {
		result.WriteString(formatPatientInfo(p))
		result.WriteString("\n---\n")
	}
	result.WriteString("\nNote: Multiple patients found. Use 'set_patient_context' with a specific patient ID to set the default.")
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result.String(),
			},
		},
	}, nil
}

func (h *Handler) ScheduleAppointment(patientID, practitionerID, dateTime, appointmentType string) (interface{}, error) {
	// Use context if IDs not provided
	patientID = h.GetContextPatientID(patientID)
	practitionerID = h.GetContextPractitionerID(practitionerID)
	
	// Check if we have required IDs
	if patientID == "" {
		return nil, fmt.Errorf("patient ID is required (no patient ID provided and none set in context)")
	}
	if practitionerID == "" {
		return nil, fmt.Errorf("practitioner ID is required (no practitioner ID provided and none set in context)")
	}
	
	// Validate patient exists
	patientExists, err := database.CheckPatientExists(h.db, patientID)
	if err != nil || !patientExists {
		return nil, fmt.Errorf("patient not found: %s", patientID)
	}
	
	// Validate practitioner exists
	practitionerExists, err := database.CheckPractitionerExists(h.db, practitionerID)
	if err != nil || !practitionerExists {
		return nil, fmt.Errorf("practitioner not found: %s", practitionerID)
	}
	
	// Parse and validate datetime
	appointmentTime, err := time.Parse(time.RFC3339, dateTime)
	if err != nil {
		return nil, fmt.Errorf("invalid datetime format (use ISO 8601): %s", dateTime)
	}
	
	// Generate new encounter ID
	encounterID := uuid.New().String()
	
	// Set default appointment type if not provided
	if appointmentType == "" {
		appointmentType = "General Consultation"
	}
	
	// Create new encounter
	encounter := &database.Encounter{
		ID:             encounterID,
		Status:         "planned",
		Class:          "ambulatory",
		TypeDisplay:    &appointmentType,
		PatientID:      patientID,
		PractitionerID: &practitionerID,
		StartDateTime:  appointmentTime.Format(time.RFC3339),
	}
	err = database.CreateEncounter(h.db, encounter)
	
	if err != nil {
		return nil, fmt.Errorf("failed to schedule appointment: %w", err)
	}
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Successfully scheduled appointment:\n\nAppointment ID: %s\nPatient ID: %s\nPractitioner ID: %s\nDate/Time: %s\nType: %s\nStatus: Scheduled",
					encounterID, patientID, practitionerID, appointmentTime.Format("2006-01-02 15:04"), appointmentType),
			},
		},
	}, nil
}

func (h *Handler) CancelAppointment(encounterID string) (interface{}, error) {
	// Check if encounter exists and is cancellable
	status, err := database.GetEncounterStatus(h.db, encounterID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("appointment not found: %s", encounterID)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	if status == "cancelled" {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Appointment %s is already cancelled", encounterID),
				},
			},
		}, nil
	}
	
	if status == "finished" {
		return nil, fmt.Errorf("cannot cancel finished appointment: %s", encounterID)
	}
	
	// Update status to cancelled
	err = database.UpdateEncounterStatus(h.db, encounterID, "cancelled")
	if err != nil {
		return nil, fmt.Errorf("failed to cancel appointment: %w", err)
	}
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Successfully cancelled appointment %s", encounterID),
			},
		},
	}, nil
}

func (h *Handler) GetMedicalHistory(patientID, category string) (interface{}, error) {
	// Use context if patient ID not provided
	patientID = h.GetContextPatientID(patientID)
	
	if patientID == "" {
		return nil, fmt.Errorf("patient ID is required (no patient ID provided and none set in context)")
	}
	
	// Validate patient exists and get name
	patientName, err := database.GetPatientName(h.db, patientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("patient not found: %s", patientID)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Medical History for %s (ID: %s)\n\n", patientName, patientID))
	
	switch category {
	case "conditions", "all":
		conditions, err := database.GetConditionsByPatientID(h.db, patientID)
		if err == nil && len(conditions) > 0 {
			result.WriteString("CONDITIONS:\n")
			for _, c := range conditions {
				result.WriteString(fmt.Sprintf("• %s (Code: %s)\n", c.Display, c.Code))
				if c.OnsetDateTime != nil {
					result.WriteString(fmt.Sprintf("  Onset: %s\n", *c.OnsetDateTime))
				}
				result.WriteString(fmt.Sprintf("  Status: %s\n", c.ClinicalStatus))
			}
			result.WriteString("\n")
		}
		if category == "conditions" {
			break
		}
		fallthrough
		
	case "medications":
		if category == "medications" || category == "all" {
			medications, err := database.GetMedicationsByPatientID(h.db, patientID)
			if err == nil && len(medications) > 0 {
				result.WriteString("MEDICATIONS:\n")
				for _, m := range medications {
					result.WriteString(fmt.Sprintf("• %s\n", m.MedicationDisplay))
					result.WriteString(fmt.Sprintf("  Status: %s\n", m.Status))
					result.WriteString(fmt.Sprintf("  Prescribed: %s\n", m.AuthoredOn))
					if m.DosageText != nil {
						result.WriteString(fmt.Sprintf("  Dosage: %s\n", *m.DosageText))
					}
				}
				result.WriteString("\n")
			}
		}
		if category == "medications" {
			break
		}
		fallthrough
		
	case "procedures":
		if category == "procedures" || category == "all" {
			procedures, err := database.GetProceduresByPatientID(h.db, patientID)
			if err == nil && len(procedures) > 0 {
				result.WriteString("PROCEDURES:\n")
				for _, p := range procedures {
					result.WriteString(fmt.Sprintf("• %s\n", p.Display))
					result.WriteString(fmt.Sprintf("  Status: %s\n", p.Status))
					if p.PerformedDateTime != nil {
						result.WriteString(fmt.Sprintf("  Performed: %s\n", *p.PerformedDateTime))
					}
				}
				result.WriteString("\n")
			}
		}
		if category == "procedures" {
			break
		}
		fallthrough
		
	case "immunizations":
		if category == "immunizations" || category == "all" {
			immunizations, err := database.GetImmunizationsByPatientID(h.db, patientID)
			if err == nil && len(immunizations) > 0 {
				result.WriteString("IMMUNIZATIONS:\n")
				for _, i := range immunizations {
					result.WriteString(fmt.Sprintf("• %s\n", i.VaccineDisplay))
					result.WriteString(fmt.Sprintf("  Date: %s\n", i.OccurrenceDateTime))
					result.WriteString(fmt.Sprintf("  Status: %s\n", i.Status))
				}
				result.WriteString("\n")
			}
		}
		if category == "immunizations" {
			break
		}
		fallthrough
		
	case "allergies":
		if category == "allergies" || category == "all" {
			allergies, err := database.GetAllergiesByPatientID(h.db, patientID)
			if err == nil && len(allergies) > 0 {
				result.WriteString("ALLERGIES:\n")
				for _, a := range allergies {
					result.WriteString(fmt.Sprintf("• %s\n", a.Display))
					result.WriteString(fmt.Sprintf("  Status: %s\n", a.ClinicalStatus))
					if a.Criticality != nil {
						result.WriteString(fmt.Sprintf("  Criticality: %s\n", *a.Criticality))
					}
				}
				result.WriteString("\n")
			}
		}
		if category == "allergies" {
			break
		}
		fallthrough
		
	case "observations":
		if category == "observations" || category == "all" {
			observations, err := database.GetObservationsByPatientID(h.db, patientID)
			if err == nil && len(observations) > 0 {
				result.WriteString("OBSERVATIONS:\n")
				for _, o := range observations {
					result.WriteString(fmt.Sprintf("• %s\n", o.Display))
					result.WriteString(fmt.Sprintf("  Category: %s\n", o.Category))
					result.WriteString(fmt.Sprintf("  Date: %s\n", o.EffectiveDateTime))
					if o.ValueQuantity != nil && o.ValueUnit != nil {
						result.WriteString(fmt.Sprintf("  Value: %.2f %s\n", *o.ValueQuantity, *o.ValueUnit))
					} else if o.ValueString != nil {
						result.WriteString(fmt.Sprintf("  Value: %s\n", *o.ValueString))
					}
					result.WriteString(fmt.Sprintf("  Status: %s\n", o.Status))
				}
				result.WriteString("\n")
			}
		}
	}
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result.String(),
			},
		},
	}, nil
}

func (h *Handler) GetMedicationInfo(medicationName string) (interface{}, error) {
	// First check database for medication
	medication, err := database.SearchMedicationByName(h.db, medicationName)
	
	var dbInfo string
	if err == nil && medication != nil {
		dbInfo = fmt.Sprintf("Found in database: %s (Code: %s)", medication.Display, medication.Code)
		if medication.Form != nil {
			dbInfo += fmt.Sprintf("\nForm: %s", *medication.Form)
		}
		dbInfo += "\n\n"
	}
	
	// Use OpenRouter to get general medication information
	prompt := fmt.Sprintf("Provide brief, factual medical information about %s including: 1) What it's used for, 2) Common dosage, 3) Important side effects or warnings. Keep response under 200 words.", medicationName)
	
	aiResponse, err := h.callOpenRouter(prompt)
	if err != nil {
		if dbInfo != "" {
			return map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": dbInfo + "Unable to fetch additional information from AI.",
					},
				},
			}, nil
		}
		return nil, fmt.Errorf("failed to get medication information: %w", err)
	}
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": dbInfo + aiResponse,
			},
		},
	}, nil
}

func (h *Handler) AnswerHealthQuestion(question string) (interface{}, error) {
	prompt := fmt.Sprintf("As a healthcare information assistant, answer this health-related question accurately and helpfully. Be conversational and don't format responses for textual responses. Be succinct.  %s", question)
	
	response, err := h.callOpenRouter(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to answer question: %w", err)
	}
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": response,
			},
		},
	}, nil
}

func (h *Handler) ProcessNaturalLanguageQuery(query string) (interface{}, error) {
	// Use function calling with OpenRouter to process natural language queries
	response, err := h.callOpenRouterWithTools(query)
	if err != nil {
		return nil, fmt.Errorf("failed to process query: %w", err)
	}
	
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": response,
			},
		},
	}, nil
}

func (h *Handler) callOpenRouter(prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": "meta-llama/llama-3.2-3b-instruct:free",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a helpful healthcare information assistant. Provide accurate, factual information. Always remind users to consult healthcare professionals for medical advice.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.3,
		"max_tokens":  500,
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Authorization", "Bearer "+h.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/eythor/mcp-server")
	req.Header.Set("X-Title", "Healthcare MCP Server")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenRouter API error: %s", string(body))
	}
	
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenRouter")
	}
	
	return result.Choices[0].Message.Content, nil
}

func (h *Handler) callOpenRouterWithTools(query string) (string, error) {
	// Get context info
	h.mu.RLock()
	hasPatientContext := h.context.PatientID != ""
	hasPractitionerContext := h.context.PractitionerID != ""
	h.mu.RUnlock()
	
	// Build required fields dynamically based on context
	scheduleRequired := []string{"datetime"}
	if !hasPatientContext {
		scheduleRequired = append(scheduleRequired, "patient_id")
	}
	if !hasPractitionerContext {
		scheduleRequired = append(scheduleRequired, "practitioner_id")
	}
	
	historyRequired := []string{}
	if !hasPatientContext {
		historyRequired = append(historyRequired, "patient_id")
	}
	
	// Define available tools for the LLM
	tools := []map[string]interface{}{
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "set_patient_context",
				"description": "Set the default patient for subsequent operations",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type": "string",
							"description": "Patient ID to set as default",
						},
					},
					"required": []string{"patient_id"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "set_practitioner_context",
				"description": "Set the default practitioner for subsequent operations",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"practitioner_id": map[string]interface{}{
							"type": "string",
							"description": "Practitioner ID to set as default",
						},
					},
					"required": []string{"practitioner_id"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "get_context",
				"description": "Get the current default patient and practitioner",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "lookup_patient",
				"description": "Look up a patient by name or ID",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type": "string",
							"description": "Patient name or ID to search for",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "get_medical_history",
				"description": "Retrieve patient medical history" + func() string {
					if hasPatientContext {
						return " (uses default patient if not specified)"
					}
					return ""
				}(),
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type": "string",
							"description": "Patient ID" + func() string {
								if hasPatientContext {
									return " (optional, uses context if not provided)"
								}
								return ""
							}(),
						},
						"category": map[string]interface{}{
							"type": "string",
							"description": "Category of history (conditions, medications, procedures, immunizations, allergies, observations, all)",
							"enum": []string{"conditions", "medications", "procedures", "immunizations", "allergies", "observations", "all"},
						},
					},
					"required": historyRequired,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "schedule_appointment",
				"description": "Schedule an appointment for a patient",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type": "string",
							"description": "Patient ID" + func() string {
								if hasPatientContext {
									return " (optional, uses context if not provided)"
								}
								return ""
							}(),
						},
						"practitioner_id": map[string]interface{}{
							"type": "string",
							"description": "Practitioner ID" + func() string {
								if hasPractitionerContext {
									return " (optional, uses context if not provided)"
								}
								return ""
							}(),
						},
						"datetime": map[string]interface{}{
							"type": "string",
							"description": "Appointment date and time (ISO 8601 format)",
						},
						"type": map[string]interface{}{
							"type": "string",
							"description": "Type of appointment",
						},
					},
					"required": scheduleRequired,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "get_medication_info",
				"description": "Get information about medications",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"medication_name": map[string]interface{}{
							"type": "string",
							"description": "Name of the medication",
						},
					},
					"required": []string{"medication_name"},
				},
			},
		},
	}

	// Build system prompt with context information
	systemPrompt := "You are a helpful healthcare assistant. You have access to patient data and can help with medical queries. Use the available tools to answer user questions accurately. Always remind users to consult healthcare professionals for medical advice."
	systemPrompt += h.GetContextInfo()
	
	reqBody := map[string]interface{}{
		// "model": "openai/gpt-4o-mini", // TODO: Decide on another model. 
		"model" : "google/gemini-2.5-flash",
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": query,
			},
		},
		"tools": tools,
		"tool_choice": "auto",
		"temperature": 0.3,
		"max_tokens": 1000,
	}

	// return log.Printf("Sending request to google/gemini-2.5-flash")
	return h.executeToolLoop(reqBody, query)
}

func (h *Handler) executeToolLoop(reqBody map[string]interface{}, originalQuery string) (string, error) {
	maxIterations := 5
	messages := reqBody["messages"].([]map[string]interface{})

	for i := 0; i < maxIterations; i++ {
		// Update messages in request
		reqBody["messages"] = messages

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+h.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("HTTP-Referer", "https://github.com/eythor/mcp-server")
		req.Header.Set("X-Title", "Healthcare MCP Server")

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("OpenRouter API error (%d): %s", resp.StatusCode, string(body))
		}

		var result struct {
			Choices []struct {
				Message struct {
					Role      string `json:"role"`
					Content   string `json:"content,omitempty"`
					ToolCalls []struct {
						ID   string `json:"id"`
						Type string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls,omitempty"`
				} `json:"message"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if len(result.Choices) == 0 {
			return "", fmt.Errorf("no response from OpenRouter")
		}

		message := result.Choices[0].Message
		
		// Add assistant message to conversation
		messages = append(messages, map[string]interface{}{
			"role":       message.Role,
			"content":    message.Content,
			"tool_calls": message.ToolCalls,
		})

		// If no tool calls, return the content
		if len(message.ToolCalls) == 0 {
			return message.Content, nil
		}

		// Execute tool calls
		for _, toolCall := range message.ToolCalls {
			result, err := h.executeTool(toolCall.Function.Name, toolCall.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error executing %s: %v", toolCall.Function.Name, err)
			}

			// Add tool result to conversation
			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": toolCall.ID,
				"content":      result,
			})
		}
	}

	return "I apologize, but I wasn't able to complete your request after multiple attempts.", nil
}

func (h *Handler) executeTool(toolName, argumentsJSON string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	switch toolName {
	case "set_patient_context":
		patientID, ok := args["patient_id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid patient_id parameter")
		}
		result, err := h.SetPatientContext(patientID)
		if err != nil {
			return "", err
		}
		return h.extractTextFromMCPResult(result), nil

	case "set_practitioner_context":
		practitionerID, ok := args["practitioner_id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid practitioner_id parameter")
		}
		result, err := h.SetPractitionerContext(practitionerID)
		if err != nil {
			return "", err
		}
		return h.extractTextFromMCPResult(result), nil

	case "get_context":
		result, err := h.GetContext()
		if err != nil {
			return "", err
		}
		return h.extractTextFromMCPResult(result), nil

	case "clear_context":
		result, err := h.ClearContext()
		if err != nil {
			return "", err
		}
		return h.extractTextFromMCPResult(result), nil

	case "lookup_patient":
		query, ok := args["query"].(string)
		if !ok {
			return "", fmt.Errorf("invalid query parameter")
		}
		result, err := h.LookupPatient(query)
		if err != nil {
			return "", err
		}
		return h.extractTextFromMCPResult(result), nil

	case "get_medical_history":
		patientID := ""
		if pid, exists := args["patient_id"].(string); exists {
			patientID = pid
		}
		category := "all"
		if cat, exists := args["category"].(string); exists {
			category = cat
		}
		result, err := h.GetMedicalHistory(patientID, category)
		if err != nil {
			return "", err
		}
		return h.extractTextFromMCPResult(result), nil

	case "schedule_appointment":
		patientID := ""
		if pid, exists := args["patient_id"].(string); exists {
			patientID = pid
		}
		practitionerID := ""
		if prid, exists := args["practitioner_id"].(string); exists {
			practitionerID = prid
		}
		datetime, ok := args["datetime"].(string)
		if !ok {
			return "", fmt.Errorf("invalid datetime parameter")
		}
		appointmentType := ""
		if t, exists := args["type"].(string); exists {
			appointmentType = t
		}
		result, err := h.ScheduleAppointment(patientID, practitionerID, datetime, appointmentType)
		if err != nil {
			return "", err
		}
		return h.extractTextFromMCPResult(result), nil

	case "get_medication_info":
		medicationName, ok := args["medication_name"].(string)
		if !ok {
			return "", fmt.Errorf("invalid medication_name parameter")
		}
		result, err := h.GetMedicationInfo(medicationName)
		if err != nil {
			return "", err
		}
		return h.extractTextFromMCPResult(result), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (h *Handler) extractTextFromMCPResult(result interface{}) string {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if content, ok := resultMap["content"].([]map[string]interface{}); ok {
			if len(content) > 0 {
				if text, ok := content[0]["text"].(string); ok {
					return text
				}
			}
		}
	}
	return fmt.Sprintf("%v", result)
}

// Helper functions
func formatPatientInfo(p database.Patient) string {
	info := fmt.Sprintf("Patient: %s %s\n", p.GivenName, p.FamilyName)
	info += fmt.Sprintf("ID: %s\n", p.ID)
	info += fmt.Sprintf("Gender: %s\n", p.Gender)
	info += fmt.Sprintf("Birth Date: %s\n", p.BirthDate)
	if p.Phone != nil {
		info += fmt.Sprintf("Phone: %s\n", *p.Phone)
	}
	if p.City != nil && p.State != nil {
		info += fmt.Sprintf("Location: %s, %s\n", *p.City, *p.State)
	}
	return info
}

