package handlers

import (
	"fmt"
	"github.com/eythor/mcp-server/internal/database"
)

// SetPatientContext sets the default patient ID in context
func (h *Handler) SetPatientContext(patientID string) (interface{}, error) {
	// Validate patient exists
	patientExists, err := database.CheckPatientExists(h.db, patientID)
	if err != nil || !patientExists {
		return nil, fmt.Errorf("patient not found: %s", patientID)
	}

	// Get patient details for confirmation
	patient, err := database.GetPatientByID(h.db, patientID)
	if err != nil {
		return nil, fmt.Errorf("error fetching patient details: %w", err)
	}

	h.mu.Lock()
	h.context.PatientID = patientID
	h.mu.Unlock()

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Context updated: Default patient set to %s %s (ID: %s)",
					patient.GivenName, patient.FamilyName, patientID),
			},
		},
	}, nil
}

// SetPractitionerContext sets the default practitioner ID in context
func (h *Handler) SetPractitionerContext(practitionerID string) (interface{}, error) {
	// Validate practitioner exists
	practitionerExists, err := database.CheckPractitionerExists(h.db, practitionerID)
	if err != nil || !practitionerExists {
		return nil, fmt.Errorf("practitioner not found: %s", practitionerID)
	}

	h.mu.Lock()
	h.context.PractitionerID = practitionerID
	h.mu.Unlock()

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Context updated: Default practitioner set to ID: %s", practitionerID),
			},
		},
	}, nil
}

// GetContext returns the current context
func (h *Handler) GetContext() (interface{}, error) {
	h.mu.RLock()
	ctx := h.context
	h.mu.RUnlock()

	message := "Current context:\n"
	
	if ctx.PatientID != "" {
		patient, err := database.GetPatientByID(h.db, ctx.PatientID)
		if err == nil {
			message += fmt.Sprintf("• Patient: %s %s (ID: %s)\n", 
				patient.GivenName, patient.FamilyName, ctx.PatientID)
		} else {
			message += fmt.Sprintf("• Patient ID: %s\n", ctx.PatientID)
		}
	} else {
		message += "• Patient: Not set\n"
	}

	if ctx.PractitionerID != "" {
		message += fmt.Sprintf("• Practitioner ID: %s\n", ctx.PractitionerID)
	} else {
		message += "• Practitioner: Not set\n"
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": message,
			},
		},
	}, nil
}

// ClearContext clears all context
func (h *Handler) ClearContext() (interface{}, error) {
	h.mu.Lock()
	h.context = Context{}
	h.mu.Unlock()

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "Context cleared. No default patient or practitioner set.",
			},
		},
	}, nil
}

// GetContextPatientID returns the patient ID from context or the provided value
func (h *Handler) GetContextPatientID(providedID string) string {
	if providedID != "" {
		return providedID
	}
	
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.context.PatientID
}

// GetContextPractitionerID returns the practitioner ID from context or the provided value
func (h *Handler) GetContextPractitionerID(providedID string) string {
	if providedID != "" {
		return providedID
	}
	
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.context.PractitionerID
}

// GetContextInfo returns formatted context information for inclusion in prompts
func (h *Handler) GetContextInfo() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if h.context.PatientID == "" && h.context.PractitionerID == "" {
		return ""
	}
	
	info := "\n\nCurrent context:"
	if h.context.PatientID != "" {
		info += fmt.Sprintf("\n- Default Patient ID: %s", h.context.PatientID)
	}
	if h.context.PractitionerID != "" {
		info += fmt.Sprintf("\n- Default Practitioner ID: %s", h.context.PractitionerID)
	}
	
	return info
}
