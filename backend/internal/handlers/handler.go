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

	// If error is not "no rows", it's a real database error
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// Patient not found by ID, try name search
	patients, err := database.SearchPatientsByName(h.db, query)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	if len(patients) == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("No patients found matching '%s'. Please check the patient ID or name and try again.", query),
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

func (h *Handler) CalculateAge(patientID string) (interface{}, error) {
	// Use context if patient ID not provided
	patientID = h.GetContextPatientID(patientID)

	if patientID == "" {
		return nil, fmt.Errorf("patient ID is required (no patient ID provided and none set in context)")
	}

	// Get patient to retrieve birth date
	patient, err := database.GetPatientByID(h.db, patientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("patient not found: %s", patientID)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	if patient.BirthDate == "" {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("No birth date available for patient %s %s (ID: %s)", patient.GivenName, patient.FamilyName, patientID),
				},
			},
		}, nil
	}

	age, err := calculateAge(patient.BirthDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate age: %w", err)
	}

	name := strings.TrimSpace(patient.GivenName + " " + patient.FamilyName)
	if name == "" || name == " " {
		name = patientID
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Patient: %s (ID: %s)\nBirth Date: %s\nAge: %d years", name, patientID, patient.BirthDate, age),
			},
		},
	}, nil
}

func (h *Handler) UpdatePatientBirthDate(patientID, birthDate string) (interface{}, error) {
	// Use context if patient ID not provided
	patientID = h.GetContextPatientID(patientID)

	if patientID == "" {
		return nil, fmt.Errorf("patient ID is required (no patient ID provided and none set in context)")
	}

	if birthDate == "" {
		return nil, fmt.Errorf("birth date is required")
	}

	// Verify patient exists
	exists, err := database.CheckPatientExists(h.db, patientID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("patient not found: %s", patientID)
	}

	// Update birth date
	err = database.UpdatePatientBirthDate(h.db, patientID, birthDate)
	if err != nil {
		return nil, fmt.Errorf("failed to update birth date: %w", err)
	}

	// Get updated patient info
	patient, err := database.GetPatientByID(h.db, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated patient: %w", err)
	}

	name := strings.TrimSpace(patient.GivenName + " " + patient.FamilyName)
	if name == "" || name == " " {
		name = patientID
	}

	age, err := calculateAge(birthDate)
	ageText := ""
	if err == nil {
		ageText = fmt.Sprintf("\nAge: %d years", age)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("✓ Birth date updated for patient %s (ID: %s)\nBirth Date: %s%s", name, patientID, birthDate, ageText),
			},
		},
	}, nil
}

func (h *Handler) AddObservation(patientID, code, display, category, status, effectiveDateTime string, valueQuantity *float64, valueUnit, valueString *string) (interface{}, error) {
	// Use context if patient ID not provided
	patientID = h.GetContextPatientID(patientID)

	if patientID == "" {
		return nil, fmt.Errorf("patient ID is required (no patient ID provided and none set in context)")
	}

	// Validate patient exists
	patientExists, err := database.CheckPatientExists(h.db, patientID)
	if err != nil || !patientExists {
		return nil, fmt.Errorf("patient not found: %s", patientID)
	}

	// Validate required fields
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}
	if display == "" {
		return nil, fmt.Errorf("display is required")
	}

	// Set defaults
	if status == "" {
		status = "final"
	}
	if category == "" {
		category = "vital-signs"
	}
	if effectiveDateTime == "" {
		effectiveDateTime = time.Now().Format(time.RFC3339)
	} else {
		// Validate datetime format
		_, err := time.Parse(time.RFC3339, effectiveDateTime)
		if err != nil {
			return nil, fmt.Errorf("invalid datetime format (use ISO 8601): %s", effectiveDateTime)
		}
	}

	// Generate new observation ID
	observationID := uuid.New().String()

	// Create observation
	observation := &database.Observation{
		ID:                observationID,
		Status:            status,
		Category:          category,
		Code:              code,
		Display:           display,
		PatientID:         patientID,
		EffectiveDateTime: &effectiveDateTime,
		ValueQuantity:     valueQuantity,
		ValueUnit:         valueUnit,
		ValueString:       valueString,
	}

	err = database.CreateObservation(h.db, observation)
	if err != nil {
		return nil, fmt.Errorf("failed to add observation: %w", err)
	}

	// Format response
	var valueText string
	if valueQuantity != nil && valueUnit != nil {
		valueText = fmt.Sprintf("%.2f %s", *valueQuantity, *valueUnit)
	} else if valueString != nil {
		valueText = *valueString
	} else {
		valueText = "N/A"
	}

	patientName, _ := database.GetPatientName(h.db, patientID)
	resultText := fmt.Sprintf("Successfully added observation:\n\nObservation ID: %s\nPatient: %s (ID: %s)\nCode: %s\nDisplay: %s\nCategory: %s\nStatus: %s\nEffective Date: %s\nValue: %s",
		observationID, patientName, patientID, code, display, category, status, effectiveDateTime, valueText)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": resultText,
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

func (h *Handler) GetClaims(patientID string) (interface{}, error) {
	// Validate patient exists
	var patientName string
	err := h.db.QueryRow("SELECT given_name || ' ' || family_name FROM patients WHERE id = ?", patientID).Scan(&patientName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("patient not found: %s", patientID)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	claims, err := h.getClaims(patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get claims: %w", err)
	}

	if len(claims) == 0 {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("No claims found for patient %s (ID: %s)", patientName, patientID),
				},
			},
		}, nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Claims for %s (ID: %s)\n\n", patientName, patientID))
	result.WriteString(fmt.Sprintf("Total Claims: %d\n\n", len(claims)))

	for i, c := range claims {
		result.WriteString(fmt.Sprintf("Claim #%d:\n", i+1))
		result.WriteString(fmt.Sprintf("  ID: %s\n", c.ID))
		result.WriteString(fmt.Sprintf("  Status: %s\n", c.Status))
		if c.Type != nil {
			result.WriteString(fmt.Sprintf("  Type: %s\n", *c.Type))
		}
		if c.Use != nil {
			result.WriteString(fmt.Sprintf("  Use: %s\n", *c.Use))
		}
		if c.Priority != nil {
			result.WriteString(fmt.Sprintf("  Priority: %s\n", *c.Priority))
		}
		if c.CreatedDateTime != nil {
			result.WriteString(fmt.Sprintf("  Created: %s\n", *c.CreatedDateTime))
		}
		if c.BillablePeriodStart != nil {
			result.WriteString(fmt.Sprintf("  Billable Period Start: %s\n", *c.BillablePeriodStart))
		}
		if c.BillablePeriodEnd != nil {
			result.WriteString(fmt.Sprintf("  Billable Period End: %s\n", *c.BillablePeriodEnd))
		}
		if c.TotalAmount != nil && c.Currency != nil {
			result.WriteString(fmt.Sprintf("  Total Amount: %s %.2f\n", *c.Currency, *c.TotalAmount))
		}
		if c.ProviderID != nil {
			result.WriteString(fmt.Sprintf("  Provider ID: %s\n", *c.ProviderID))
		}
		result.WriteString("\n")
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

func (h *Handler) DetermineApixabanDose(patientID string) (interface{}, error) {
	// Use context if patient ID not provided
	patientID = h.GetContextPatientID(patientID)

	if patientID == "" {
		return nil, fmt.Errorf("patient ID is required (no patient ID provided and none set in context)")
	}

	// Get patient to retrieve birth date and calculate age
	patient, err := database.GetPatientByID(h.db, patientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("patient not found: %s", patientID)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Calculate age
	var age int
	ageMet := false
	if patient.BirthDate != "" {
		age, err = calculateAge(patient.BirthDate)
		if err == nil {
			ageMet = age >= 80
		}
	}

	// Get all observations for the patient
	observations, err := database.GetObservationsByPatientID(h.db, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get observations: %w", err)
	}

	// Find most recent body weight observation
	var bodyWeight *float64
	bodyWeightMet := false
	for _, obs := range observations {
		displayLower := strings.ToLower(obs.Display)
		if (strings.Contains(displayLower, "weight") || strings.Contains(displayLower, "body weight")) &&
			obs.ValueQuantity != nil && obs.ValueUnit != nil {
			// Check if unit is kg or convert if needed
			unitLower := strings.ToLower(*obs.ValueUnit)
			weight := *obs.ValueQuantity
			if strings.Contains(unitLower, "kg") {
				bodyWeight = &weight
				bodyWeightMet = weight <= 60.0
				break
			} else if strings.Contains(unitLower, "lb") || strings.Contains(unitLower, "pound") {
				// Convert pounds to kg (1 lb = 0.453592 kg)
				weightKg := weight * 0.453592
				bodyWeight = &weightKg
				bodyWeightMet = weightKg <= 60.0
				break
			}
		}
	}

	// Find most recent serum creatinine observation
	var serumCreatinine *float64
	creatinineMet := false
	for _, obs := range observations {
		displayLower := strings.ToLower(obs.Display)
		if (strings.Contains(displayLower, "creatinine") || strings.Contains(displayLower, "serum creatinine")) &&
			obs.ValueQuantity != nil && obs.ValueUnit != nil {
			unitLower := strings.ToLower(*obs.ValueUnit)
			creatinine := *obs.ValueQuantity
			// Check if unit is mg/dL or convert if needed
			if strings.Contains(unitLower, "mg/dl") || strings.Contains(unitLower, "mg/dL") {
				serumCreatinine = &creatinine
				creatinineMet = creatinine >= 1.5
				break
			} else if strings.Contains(unitLower, "umol/l") || strings.Contains(unitLower, "μmol/l") {
				// Convert μmol/L to mg/dL (1 mg/dL = 88.4 μmol/L)
				creatinineMgDl := creatinine / 88.4
				serumCreatinine = &creatinineMgDl
				creatinineMet = creatinineMgDl >= 1.5
				break
			}
		}
	}

	// Count how many conditions are met
	conditionsMet := 0
	var conditions []string
	if ageMet {
		conditionsMet++
		conditions = append(conditions, fmt.Sprintf("Age ≥80 years: ✓ (%d years)", age))
	} else if patient.BirthDate != "" {
		conditions = append(conditions, fmt.Sprintf("Age ≥80 years: ✗ (%d years)", age))
	} else {
		conditions = append(conditions, "Age ≥80 years: ✗ (birth date not available)")
	}

	if bodyWeightMet {
		conditionsMet++
		conditions = append(conditions, fmt.Sprintf("Body weight ≤60 kg: ✓ (%.2f kg)", *bodyWeight))
	} else if bodyWeight != nil {
		conditions = append(conditions, fmt.Sprintf("Body weight ≤60 kg: ✗ (%.2f kg)", *bodyWeight))
	} else {
		conditions = append(conditions, "Body weight ≤60 kg: ✗ (not found in observations)")
	}

	if creatinineMet {
		conditionsMet++
		conditions = append(conditions, fmt.Sprintf("Serum creatinine ≥1.5 mg/dL: ✓ (%.2f mg/dL)", *serumCreatinine))
	} else if serumCreatinine != nil {
		conditions = append(conditions, fmt.Sprintf("Serum creatinine ≥1.5 mg/dL: ✗ (%.2f mg/dL)", *serumCreatinine))
	} else {
		conditions = append(conditions, "Serum creatinine ≥1.5 mg/dL: ✗ (not found in observations)")
	}

	// Determine dose: half dose if 2 out of 3 conditions are met
	dose := "Full dose"
	if conditionsMet >= 2 {
		dose = "Half dose"
	}

	patientName, _ := database.GetPatientName(h.db, patientID)
	resultText := fmt.Sprintf("Apixaban Dose Determination for %s (ID: %s)\n\n", patientName, patientID)
	resultText += fmt.Sprintf("Conditions evaluated:\n")
	for _, cond := range conditions {
		resultText += fmt.Sprintf("• %s\n", cond)
	}
	resultText += fmt.Sprintf("\nConditions met: %d out of 3\n", conditionsMet)
	resultText += fmt.Sprintf("\nRecommendation: %s of Apixaban", dose)
	if conditionsMet >= 2 {
		resultText += "\n\nReason: 2 or more dose reduction criteria are met (age ≥80 years, body weight ≤60 kg, or serum creatinine ≥1.5 mg/dL)."
	} else {
		resultText += "\n\nReason: Less than 2 dose reduction criteria are met."
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": resultText,
			},
		},
	}, nil
}

func (h *Handler) GetMedicalGuidelines(query string) (interface{}, error) {
	// Build a comprehensive prompt for medical guidelines and information
	systemContext := `You are a medical information assistant providing evidence-based information about:
- Clinical guidelines and best practices
- Medication information (uses, dosages, interactions, contraindications)
- Treatment protocols and recommendations
- Diagnostic criteria and procedures
- Medical terminology and conditions

Important guidelines:
1. Provide accurate, evidence-based information
2. Include typical dosage ranges when discussing medications
3. Mention important contraindications and warnings
4. Reference standard guidelines (e.g., WHO, CDC, FDA) when applicable
5. Always remind users to consult healthcare professionals for personal medical advice
6. Be comprehensive but concise (aim for 300-500 words for detailed queries)`

	prompt := fmt.Sprintf("%s\n\nUser Query: %s", systemContext, query)

	// Use OpenRouter with a more capable model for medical information
	reqBody := map[string]interface{}{
		"model": "google/gemini-2.0-flash-exp:free", // Using a more capable model for medical info
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a knowledgeable medical information assistant. Provide accurate, evidence-based medical information while always reminding users to consult healthcare professionals for personal medical decisions.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.2,  // Lower temperature for factual accuracy
		"max_tokens":  1500, // Allow longer responses for detailed medical info
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+h.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/eythor/mcp-server")
	req.Header.Set("X-Title", "Healthcare MCP Server")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	response := result.Choices[0].Message.Content

	// Add context information if available
	if h.context.PatientID != "" {
		response += fmt.Sprintf("\n\nNote: This information is general medical guidance. For patient-specific recommendations for Patient ID %s, please consult with the treating physician.", h.context.PatientID)
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

func (h *Handler) ProcessNaturalLanguageQuery(query string, practitionerID string) (interface{}, error) {
	// Use function calling with OpenRouter to process natural language queries
	response, err := h.callOpenRouterWithTools(query, practitionerID)
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

func (h *Handler) callOpenRouterWithTools(query string, practitionerID string) (string, error) {
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

	ageRequired := []string{}
	if !hasPatientContext {
		ageRequired = append(ageRequired, "patient_id")
	}

	// Define available tools for the LLM
	tools := []map[string]interface{}{
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "set_patient_context",
				"description": "Set the default patient for subsequent operations",
				"parameters": map[string]interface{}{
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
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "set_practitioner_context",
				"description": "Set the default practitioner for subsequent operations",
				"parameters": map[string]interface{}{
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
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_context",
				"description": "Get the current default patient and practitioner",
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "lookup_patient",
				"description": "Look up a patient by name or ID. Returns patient information including: name, patient ID, gender, birth_date (date of birth), age (calculated automatically), phone number, and location. Birth date and age are always included when available in the patient record.",
				"parameters": map[string]interface{}{
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
							"type":        "string",
							"description": "Category of history (conditions, medications, procedures, immunizations, allergies, observations, all)",
							"enum":        []string{"conditions", "medications", "procedures", "immunizations", "allergies", "observations", "all"},
						},
					},
					"required": historyRequired,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "schedule_appointment",
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
							"type":        "string",
							"description": "Appointment date and time (ISO 8601 format)",
						},
						"type": map[string]interface{}{
							"type":        "string",
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
				"name":        "get_medication_info",
				"description": "Get information about medications",
				"parameters": map[string]interface{}{
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
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_claims",
				"description": "Retrieve insurance claims for a patient",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"patient_id": map[string]interface{}{
							"type":        "string",
							"description": "Patient ID",
						},
					},
					"required": []string{"patient_id"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "add_observation",
				"description": "Add an observation record for a patient (e.g., vital signs, lab results, measurements)" + func() string {
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
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "calculate_age",
				"description": "Calculate the age of a patient from their birth date. Returns the patient's current age in years based on their birth date stored in the database." + func() string {
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
					},
					"required": ageRequired,
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "update_patient_birth_date",
				"description": "Update a patient's birth date in the database" + func() string {
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
						"birth_date": map[string]interface{}{
							"type":        "string",
							"description": "Birth date in YYYY-MM-DD format (ISO 8601)",
						},
					},
					"required": []string{"birth_date"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "get_medical_guidelines",
				"description": "Get comprehensive medical guidelines, dosages, treatment protocols, and clinical best practices",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Medical query (e.g., 'diabetes management', 'antibiotic dosing', 'hypertension guidelines')",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name": "determine_apixaban_dose",
				"description": "Determine whether to give half or full dose of Apixaban based on patient criteria. Half dose is recommended if 2 out of 3 conditions are met: age ≥80 years, body weight ≤60 kg, or serum creatinine ≥1.5 mg/dL." + func() string {
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
					},
					"required": historyRequired,
				},
			},
		},
	}

	// Build system prompt with context information
	systemPrompt := "You are a helpful healthcare assistant. You have access to patient data and can help with medical queries. Use the available tools to answer user questions accurately. Always reply in english. CRITICAL: Keep responses extremely brief and concise - aim for 2-4 sentences maximum. Your responses will be converted to audio, so brevity is essential. For medical history queries, provide a high-level summary of key conditions, recent procedures, and current medications - do NOT list every detail. Focus on the most important and recent information only."
	systemPrompt += h.GetContextInfo()

	reqBody := map[string]interface{}{
		"model": "google/gemini-2.5-flash",
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
		"tools":       tools,
		"tool_choice": "auto",
		"temperature": 0.3,
		"max_tokens":  1000,
	}

	// return log.Printf("Sending request to google/gemini-2.5-flash")
	return h.executeToolLoop(reqBody, query, practitionerID)
}

func (h *Handler) executeToolLoop(reqBody map[string]interface{}, originalQuery string, practitionerID string) (string, error) {
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
						ID       string `json:"id"`
						Type     string `json:"type"`
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
			result, err := h.executeTool(toolCall.Function.Name, toolCall.Function.Arguments, practitionerID)
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

func (h *Handler) executeTool(toolName, argumentsJSON string, defaultPractitionerID string) (string, error) {
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
		return h.ExtractTextFromMCPResult(result), nil

	case "set_practitioner_context":
		practitionerID, ok := args["practitioner_id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid practitioner_id parameter")
		}
		result, err := h.SetPractitionerContext(practitionerID)
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	case "get_context":
		result, err := h.GetContext()
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	case "clear_context":
		result, err := h.ClearContext()
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	case "lookup_patient":
		query, ok := args["query"].(string)
		if !ok {
			return "", fmt.Errorf("invalid query parameter")
		}
		result, err := h.LookupPatient(query)
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

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
		return h.ExtractTextFromMCPResult(result), nil

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
		return h.ExtractTextFromMCPResult(result), nil

	case "get_medication_info":
		medicationName, ok := args["medication_name"].(string)
		if !ok {
			return "", fmt.Errorf("invalid medication_name parameter")
		}
		result, err := h.GetMedicationInfo(medicationName)
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	case "get_claims":
		patientID, ok := args["patient_id"].(string)
		if !ok {
			return "", fmt.Errorf("invalid patient_id parameter")
		}
		result, err := h.GetClaims(patientID)
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	case "add_observation":
		patientID := ""
		if pid, exists := args["patient_id"].(string); exists {
			patientID = pid
		}
		code, ok := args["code"].(string)
		if !ok {
			return "", fmt.Errorf("invalid code parameter")
		}
		display, ok := args["display"].(string)
		if !ok {
			return "", fmt.Errorf("invalid display parameter")
		}
		category := ""
		if cat, exists := args["category"].(string); exists {
			category = cat
		}
		status := ""
		if st, exists := args["status"].(string); exists {
			status = st
		}
		effectiveDateTime := ""
		if edt, exists := args["effective_datetime"].(string); exists {
			effectiveDateTime = edt
		}
		var valueQuantity *float64
		if vq, exists := args["value_quantity"]; exists {
			if vqFloat, ok := vq.(float64); ok {
				valueQuantity = &vqFloat
			}
		}
		var valueUnit *string
		if vu, exists := args["value_unit"].(string); exists {
			valueUnit = &vu
		}
		var valueString *string
		if vs, exists := args["value_string"].(string); exists {
			valueString = &vs
		}
		result, err := h.AddObservation(patientID, code, display, category, status, effectiveDateTime, valueQuantity, valueUnit, valueString)
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	case "calculate_age":
		patientID := ""
		if pid, exists := args["patient_id"].(string); exists {
			patientID = pid
		}
		result, err := h.CalculateAge(patientID)
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	case "update_patient_birth_date":
		patientID := ""
		if pid, exists := args["patient_id"].(string); exists {
			patientID = pid
		}
		birthDate, ok := args["birth_date"].(string)
		if !ok {
			return "", fmt.Errorf("invalid birth_date parameter")
		}
		result, err := h.UpdatePatientBirthDate(patientID, birthDate)
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	case "get_medical_guidelines":
		query, ok := args["query"].(string)
		if !ok {
			return "", fmt.Errorf("invalid query parameter")
		}
		result, err := h.GetMedicalGuidelines(query)
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	case "determine_apixaban_dose":
		patientID := ""
		if pid, exists := args["patient_id"].(string); exists {
			patientID = pid
		}
		result, err := h.DetermineApixabanDose(patientID)
		if err != nil {
			return "", err
		}
		return h.ExtractTextFromMCPResult(result), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (h *Handler) ExtractTextFromMCPResult(result interface{}) string {
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
func calculateAge(birthDateStr string) (int, error) {
	if birthDateStr == "" {
		return 0, fmt.Errorf("birth date is empty")
	}

	// Try common date formats
	formats := []string{
		"2006-01-02",           // YYYY-MM-DD (ISO date)
		"2006-01-02T15:04:05Z", // ISO 8601 with time
		"2006-01-02T15:04:05-07:00",
		"01/02/2006", // MM/DD/YYYY
		"02/01/2006", // DD/MM/YYYY
	}

	var birthDate time.Time
	var err error
	for _, format := range formats {
		birthDate, err = time.Parse(format, birthDateStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return 0, fmt.Errorf("unable to parse birth date: %s", birthDateStr)
	}

	now := time.Now()
	age := now.Year() - birthDate.Year()

	// Adjust if birthday hasn't occurred this year
	if now.YearDay() < birthDate.YearDay() {
		age--
	}

	return age, nil
}

func formatPatientInfo(p database.Patient) string {
	var info strings.Builder

	// Always start with a clear confirmation that patient was found
	info.WriteString(fmt.Sprintf("Patient found:\n"))

	// Handle name display - show ID if name is empty
	name := strings.TrimSpace(p.GivenName + " " + p.FamilyName)
	if name == "" || name == " " {
		info.WriteString(fmt.Sprintf("Patient ID: %s\n", p.ID))
	} else {
		info.WriteString(fmt.Sprintf("Name: %s\n", name))
		info.WriteString(fmt.Sprintf("Patient ID: %s\n", p.ID))
	}

	// Always show birth date prominently if available, and calculate age
	// Explicitly state when birth date is not available so LLM knows it's missing
	if p.BirthDate != "" {
		info.WriteString(fmt.Sprintf("Birth Date: %s\n", p.BirthDate))
		age, err := calculateAge(p.BirthDate)
		if err == nil {
			info.WriteString(fmt.Sprintf("Age: %d years\n", age))
		}
	} else {
		info.WriteString(fmt.Sprintf("Birth Date: Not available\n"))
	}

	if p.Gender != "" && p.Gender != "unknown" {
		info.WriteString(fmt.Sprintf("Gender: %s\n", p.Gender))
	}
	if p.Phone != nil && *p.Phone != "" {
		info.WriteString(fmt.Sprintf("Phone: %s\n", *p.Phone))
	}
	if p.City != nil && p.State != nil && (*p.City != "" || *p.State != "") {
		info.WriteString(fmt.Sprintf("Location: %s, %s\n", *p.City, *p.State))
	}
	return info.String()
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

func (h *Handler) getClaims(patientID string) ([]database.Claim, error) {
	rows, err := h.db.Query(`
		SELECT id, status, type, use, patient_id, provider_id, priority, 
		       created_datetime, billable_period_start, billable_period_end, 
		       total_amount, currency
		FROM claims
		WHERE patient_id = ?
		ORDER BY created_datetime DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var claims []database.Claim
	for rows.Next() {
		var c database.Claim
		err := rows.Scan(
			&c.ID, &c.Status, &c.Type, &c.Use, &c.PatientID, &c.ProviderID,
			&c.Priority, &c.CreatedDateTime, &c.BillablePeriodStart,
			&c.BillablePeriodEnd, &c.TotalAmount, &c.Currency,
		)
		if err != nil {
			continue
		}
		claims = append(claims, c)
	}
	return claims, nil
}
