package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eythor/mcp-server/internal/database"
	"github.com/google/uuid"
)

type Handler struct {
	db     *sql.DB
	apiKey string
}

func NewHandler(db *sql.DB, apiKey string) *Handler {
	return &Handler{
		db:     db,
		apiKey: apiKey,
	}
}

func (h *Handler) LookupPatient(query string) (interface{}, error) {
	query = strings.TrimSpace(query)
	
	var patient database.Patient
	err := h.db.QueryRow(`
		SELECT id, given_name, family_name, gender, birth_date, phone, city, state
		FROM patients
		WHERE id = ?
	`, query).Scan(
		&patient.ID, &patient.GivenName, &patient.FamilyName,
		&patient.Gender, &patient.BirthDate, &patient.Phone,
		&patient.City, &patient.State,
	)
	
	if err == nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": formatPatientInfo(patient),
				},
			},
		}, nil
	}
	
	// Try name search
	searchQuery := "%" + query + "%"
	rows, err := h.db.Query(`
		SELECT id, given_name, family_name, gender, birth_date, phone, city, state
		FROM patients
		WHERE given_name LIKE ? OR family_name LIKE ? OR (given_name || ' ' || family_name) LIKE ?
	`, searchQuery, searchQuery, searchQuery)
	
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()
	
	var patients []database.Patient
	for rows.Next() {
		var p database.Patient
		err := rows.Scan(
			&p.ID, &p.GivenName, &p.FamilyName,
			&p.Gender, &p.BirthDate, &p.Phone,
			&p.City, &p.State,
		)
		if err != nil {
			continue
		}
		patients = append(patients, p)
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
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d patient(s) matching '%s':\n\n", len(patients), query))
	for _, p := range patients {
		result.WriteString(formatPatientInfo(p))
		result.WriteString("\n---\n")
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

func (h *Handler) ScheduleAppointment(patientID, practitionerID, dateTime, appointmentType string) (interface{}, error) {
	// Validate patient exists
	var patientExists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM patients WHERE id = ?)", patientID).Scan(&patientExists)
	if err != nil || !patientExists {
		return nil, fmt.Errorf("patient not found: %s", patientID)
	}
	
	// Validate practitioner exists
	var practitionerExists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM practitioners WHERE id = ?)", practitionerID).Scan(&practitionerExists)
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
	
	// Insert new encounter
	_, err = h.db.Exec(`
		INSERT INTO encounters (
			id, resource_type, status, class, type_display,
			patient_id, practitioner_id, start_datetime
		) VALUES (?, 'Encounter', 'planned', 'ambulatory', ?, ?, ?, ?)
	`, encounterID, appointmentType, patientID, practitionerID, appointmentTime)
	
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
	var status string
	err := h.db.QueryRow("SELECT status FROM encounters WHERE id = ?", encounterID).Scan(&status)
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
	_, err = h.db.Exec("UPDATE encounters SET status = 'cancelled' WHERE id = ?", encounterID)
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
	// Validate patient exists
	var patientName string
	err := h.db.QueryRow("SELECT given_name || ' ' || family_name FROM patients WHERE id = ?", patientID).Scan(&patientName)
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
		conditions, err := h.getConditions(patientID)
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
			medications, err := h.getMedications(patientID)
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
			procedures, err := h.getProcedures(patientID)
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
			immunizations, err := h.getImmunizations(patientID)
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
			allergies, err := h.getAllergies(patientID)
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
	var medication struct {
		Code    string
		Display string
		Form    *string
	}
	
	err := h.db.QueryRow(`
		SELECT code, display, form
		FROM medications
		WHERE display LIKE ?
		LIMIT 1
	`, "%"+medicationName+"%").Scan(&medication.Code, &medication.Display, &medication.Form)
	
	var dbInfo string
	if err == nil {
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
	prompt := fmt.Sprintf("As a healthcare information assistant, answer this health-related question accurately and helpfully. Always remind users to consult healthcare professionals for medical advice. Question: %s", question)
	
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
	// Define available tools for the LLM
	tools := []map[string]interface{}{
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
				"description": "Retrieve patient medical history",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type": "string",
							"description": "Patient ID",
						},
						"category": map[string]interface{}{
							"type": "string",
							"description": "Category of history (conditions, medications, procedures, immunizations, allergies, all)",
							"enum": []string{"conditions", "medications", "procedures", "immunizations", "allergies", "all"},
						},
					},
					"required": []string{"patient_id"},
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
							"description": "Patient ID",
						},
						"practitioner_id": map[string]interface{}{
							"type": "string",
							"description": "Practitioner ID",
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
					"required": []string{"patient_id", "practitioner_id", "datetime"},
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

	reqBody := map[string]interface{}{
		"model": "openai/gpt-4o-mini", // TODO: Decide on another model. 
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				// TODO:L We need to fine tune this later
				"content": "You are a helpful healthcare assistant. You have access to patient data and can help with medical queries. Use the available tools to answer user questions accurately. Always remind users to consult healthcare professionals for medical advice.",
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
		patientID, ok := args["patient_id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid patient_id parameter")
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
		patientID, ok := args["patient_id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid patient_id parameter")
		}
		practitionerID, ok := args["practitioner_id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid practitioner_id parameter")
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

func (h *Handler) getConditions(patientID string) ([]database.Condition, error) {
	rows, err := h.db.Query(`
		SELECT id, clinical_status, code, display, patient_id, onset_datetime
		FROM conditions
		WHERE patient_id = ?
		ORDER BY onset_datetime DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var conditions []database.Condition
	for rows.Next() {
		var c database.Condition
		err := rows.Scan(&c.ID, &c.ClinicalStatus, &c.Code, &c.Display, &c.PatientID, &c.OnsetDateTime)
		if err != nil {
			continue
		}
		conditions = append(conditions, c)
	}
	return conditions, nil
}

func (h *Handler) getMedications(patientID string) ([]database.MedicationRequest, error) {
	rows, err := h.db.Query(`
		SELECT id, status, medication_display, patient_id, authored_on, dosage_text
		FROM medication_requests
		WHERE patient_id = ?
		ORDER BY authored_on DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var medications []database.MedicationRequest
	for rows.Next() {
		var m database.MedicationRequest
		err := rows.Scan(&m.ID, &m.Status, &m.MedicationDisplay, &m.PatientID, &m.AuthoredOn, &m.DosageText)
		if err != nil {
			continue
		}
		medications = append(medications, m)
	}
	return medications, nil
}

func (h *Handler) getProcedures(patientID string) ([]database.Procedure, error) {
	rows, err := h.db.Query(`
		SELECT id, status, display, patient_id, performed_datetime
		FROM procedures
		WHERE patient_id = ?
		ORDER BY performed_datetime DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var procedures []database.Procedure
	for rows.Next() {
		var p database.Procedure
		err := rows.Scan(&p.ID, &p.Status, &p.Display, &p.PatientID, &p.PerformedDateTime)
		if err != nil {
			continue
		}
		procedures = append(procedures, p)
	}
	return procedures, nil
}

func (h *Handler) getImmunizations(patientID string) ([]database.Immunization, error) {
	rows, err := h.db.Query(`
		SELECT id, status, vaccine_display, patient_id, occurrence_datetime
		FROM immunizations
		WHERE patient_id = ?
		ORDER BY occurrence_datetime DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var immunizations []database.Immunization
	for rows.Next() {
		var i database.Immunization
		err := rows.Scan(&i.ID, &i.Status, &i.VaccineDisplay, &i.PatientID, &i.OccurrenceDateTime)
		if err != nil {
			continue
		}
		immunizations = append(immunizations, i)
	}
	return immunizations, nil
}

func (h *Handler) getAllergies(patientID string) ([]database.AllergyIntolerance, error) {
	rows, err := h.db.Query(`
		SELECT id, clinical_status, display, patient_id, criticality
		FROM allergy_intolerances
		WHERE patient_id = ?
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var allergies []database.AllergyIntolerance
	for rows.Next() {
		var a database.AllergyIntolerance
		err := rows.Scan(&a.ID, &a.ClinicalStatus, &a.Display, &a.PatientID, &a.Criticality)
		if err != nil {
			continue
		}
		allergies = append(allergies, a)
	}
	return allergies, nil
}
