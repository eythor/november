package handlers

import (
	"fmt"
	"time"

	"github.com/eythor/mcp-server/internal/database"
	"github.com/eythor/mcp-server/internal/debug"
)

// fetchPatientMedicalSummary fetches and formats patient medical data for context
func (h *Handler) fetchPatientMedicalSummary(patientID string) (*PatientMedicalSummary, error) {
	debug.Verbose("Fetching medical summary for patient: %s", patientID)
	
	summary := &PatientMedicalSummary{
		LastUpdated: time.Now().Format(time.RFC3339),
	}
	
	// Get patient demographics
	patient, err := database.GetPatientByID(h.db, patientID)
	if err != nil {
		return nil, fmt.Errorf("error fetching patient: %w", err)
	}
	
	// Format demographics
	age := "Unknown"
	if patient.BirthDate != "" {
		if calcAge, err := calculateAge(patient.BirthDate); err == nil {
			age = fmt.Sprintf("%d years", calcAge)
		}
	}
	summary.Demographics = fmt.Sprintf("%s %s, %s, Age: %s", 
		patient.GivenName, patient.FamilyName, patient.Gender, age)
	
	// Get active conditions
	conditions, err := database.GetConditionsByPatientID(h.db, patientID)
	if err == nil {
		for _, c := range conditions {
			if c.ClinicalStatus == "active" || c.ClinicalStatus == "" {
				conditionText := c.Display
				if c.OnsetDateTime != nil && *c.OnsetDateTime != "" {
					conditionText += fmt.Sprintf(" (since %s)", (*c.OnsetDateTime)[:10])
				}
				summary.ActiveConditions = append(summary.ActiveConditions, conditionText)
			}
		}
	}
	
	// Get current medications
	medications, err := database.GetMedicationsByPatientID(h.db, patientID)
	if err == nil {
		for _, m := range medications {
			if m.Status == "active" || m.Status == "" {
				medText := m.MedicationDisplay
				if m.DosageText != nil && *m.DosageText != "" {
					medText += fmt.Sprintf(" - %s", *m.DosageText)
				}
				summary.CurrentMedications = append(summary.CurrentMedications, medText)
			}
		}
	}
	
	// Get recent observations (last 5)
	observations, err := database.GetObservationsByPatientID(h.db, patientID)
	if err == nil {
		count := 0
		for _, o := range observations {
			if count >= 5 {
				break
			}
			obsText := o.Display
			if o.ValueQuantity != nil && o.ValueUnit != nil {
				obsText += fmt.Sprintf(": %.2f %s", *o.ValueQuantity, *o.ValueUnit)
			} else if o.ValueString != nil {
				obsText += fmt.Sprintf(": %s", *o.ValueString)
			}
			if o.EffectiveDateTime != nil {
				obsText += fmt.Sprintf(" (%s)", (*o.EffectiveDateTime)[:10])
			}
			summary.RecentObservations = append(summary.RecentObservations, obsText)
			count++
		}
	}
	
	// Get allergies
	allergies, err := database.GetAllergiesByPatientID(h.db, patientID)
	if err == nil {
		for _, a := range allergies {
			if a.ClinicalStatus == "active" || a.ClinicalStatus == "" {
				allergyText := a.Display
				if a.Criticality != nil && *a.Criticality != "" {
					allergyText += fmt.Sprintf(" (%s)", *a.Criticality)
				}
				summary.Allergies = append(summary.Allergies, allergyText)
			}
		}
	}
	
	// Get recent encounters
	encounters, err := database.GetEncountersByPatientID(h.db, patientID)
	if err == nil {
		summary.TotalEncounters = len(encounters)
		
		// Format last encounter date if available
		if len(encounters) > 0 {
			lastEnc := encounters[0]
			if lastEnc.StartDateTime != "" {
				// Parse and format the date
				if len(lastEnc.StartDateTime) >= 10 {
					summary.LastEncounter = lastEnc.StartDateTime[:10]
				} else {
					summary.LastEncounter = lastEnc.StartDateTime
				}
			}
		}
		
		// Get recent encounters (last 5)
		count := 0
		for _, e := range encounters {
			if count >= 5 {
				break
			}
			
			encText := ""
			// Format encounter type/class
			if e.TypeDisplay != nil && *e.TypeDisplay != "" {
				encText = *e.TypeDisplay
			} else {
				encText = e.Class
			}
			
			// Add date
			if e.StartDateTime != "" {
				date := e.StartDateTime
				if len(date) >= 10 {
					date = date[:10]
				}
				encText += fmt.Sprintf(" on %s", date)
			}
			
			// Add status if not finished
			if e.Status != "finished" && e.Status != "" {
				encText += fmt.Sprintf(" (%s)", e.Status)
			}
			
			// Add duration if end time exists
			if e.EndDateTime != nil && *e.EndDateTime != "" && e.StartDateTime != "" {
				// For simplicity, just note if it was same day or multi-day
				if len(e.StartDateTime) >= 10 && len(*e.EndDateTime) >= 10 {
					startDate := e.StartDateTime[:10]
					endDate := (*e.EndDateTime)[:10]
					if startDate != endDate {
						encText += " (multi-day)"
					}
				}
			}
			
			summary.RecentEncounters = append(summary.RecentEncounters, encText)
			count++
		}
	}
	
	debug.Verbose("Medical summary fetched: %d conditions, %d medications, %d observations, %d allergies, %d encounters",
		len(summary.ActiveConditions), len(summary.CurrentMedications), 
		len(summary.RecentObservations), len(summary.Allergies), len(summary.RecentEncounters))
	
	return summary, nil
}

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
	
	// Fetch medical summary
	medicalSummary, err := h.fetchPatientMedicalSummary(patientID)
	if err != nil {
		debug.Error("Failed to fetch medical summary: %v", err)
		// Continue without medical summary
		medicalSummary = nil
	}

	h.mu.Lock()
	h.context.PatientID = patientID
	h.context.PatientSummary = medicalSummary
	h.context.LastResponse = "" // Clear last response when changing patient
	h.mu.Unlock()
	
	debug.Log("Patient context set: %s %s (ID: %s), medical summary loaded: %v, last response cleared",
		patient.GivenName, patient.FamilyName, patientID, medicalSummary != nil)

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Context updated: Current patient set to %s %s (ID: %s)",
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
				"text": fmt.Sprintf("Context updated: Current practitioner set to ID: %s", practitionerID),
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
		
		// Show encounter summary if available
		if ctx.PatientSummary != nil {
			if ctx.PatientSummary.LastEncounter != "" {
				message += fmt.Sprintf("  - Last visit: %s", ctx.PatientSummary.LastEncounter)
				if ctx.PatientSummary.TotalEncounters > 0 {
					message += fmt.Sprintf(" (Total: %d visits)\n", ctx.PatientSummary.TotalEncounters)
				} else {
					message += "\n"
				}
			}
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
	
	debug.Log("Context cleared, including patient medical summary and last response")

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "Context cleared. No current patient or practitioner set.",
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

// SetLastResponse updates the last response in context
func (h *Handler) SetLastResponse(response string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.context.LastResponse = response
	debug.Verbose("Last response updated in context (length: %d)", len(response))
}

// GetContextInfo returns formatted context information for inclusion in prompts
func (h *Handler) GetContextInfo() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Always include current timestamp
	currentTime := time.Now().Format(time.RFC3339)
	info := fmt.Sprintf("\n\nCurrent date and time: %s", currentTime)
	
	// Include last response if available (for conversation continuity)
	if h.context.LastResponse != "" {
		info += "\n\n**Previous Response:**"
		// Truncate if too long to avoid context bloat
		if len(h.context.LastResponse) > 500 {
			info += fmt.Sprintf("\n%s... (truncated)", h.context.LastResponse[:500])
		} else {
			info += fmt.Sprintf("\n%s", h.context.LastResponse)
		}
	}

	if h.context.PatientID != "" || h.context.PractitionerID != "" {
		info += "\n\nCurrent context:"
		if h.context.PatientID != "" {
			info += fmt.Sprintf("\n- Current Patient ID: %s", h.context.PatientID)
			
			// Include patient medical summary if available
			if h.context.PatientSummary != nil {
				info += "\n\n**Patient Medical Summary:**"
				info += fmt.Sprintf("\n- Demographics: %s", h.context.PatientSummary.Demographics)
				
				// Encounter information
				if h.context.PatientSummary.LastEncounter != "" {
					info += fmt.Sprintf("\n- Last Visit: %s", h.context.PatientSummary.LastEncounter)
				}
				if h.context.PatientSummary.TotalEncounters > 0 {
					info += fmt.Sprintf(" (Total visits: %d)", h.context.PatientSummary.TotalEncounters)
				}
				
				if len(h.context.PatientSummary.RecentEncounters) > 0 {
					info += "\n- Recent Encounters:"
					for _, enc := range h.context.PatientSummary.RecentEncounters {
						info += fmt.Sprintf("\n  • %s", enc)
					}
				}
				
				if len(h.context.PatientSummary.ActiveConditions) > 0 {
					info += "\n- Active Conditions:"
					for _, condition := range h.context.PatientSummary.ActiveConditions {
						info += fmt.Sprintf("\n  • %s", condition)
					}
				}
				
				if len(h.context.PatientSummary.CurrentMedications) > 0 {
					info += "\n- Current Medications:"
					for _, med := range h.context.PatientSummary.CurrentMedications {
						info += fmt.Sprintf("\n  • %s", med)
					}
				}
				
				if len(h.context.PatientSummary.Allergies) > 0 {
					info += "\n- Allergies:"
					for _, allergy := range h.context.PatientSummary.Allergies {
						info += fmt.Sprintf("\n  • %s", allergy)
					}
				}
				
				if len(h.context.PatientSummary.RecentObservations) > 0 {
					info += "\n- Recent Observations:"
					for _, obs := range h.context.PatientSummary.RecentObservations {
						info += fmt.Sprintf("\n  • %s", obs)
					}
				}
			}
		}
		if h.context.PractitionerID != "" {
			info += fmt.Sprintf("\n- Current Practitioner ID: %s", h.context.PractitionerID)
		}
	}

	return info
}
