package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/eythor/mcp-server/internal/debug"
)

func GetPatientByID(db *sql.DB, id string) (*Patient, error) {
	debug.Verbose("GetPatientByID called with id: %s", id)
	var patient Patient
	var birthDate, phone, city, state sql.NullString

	query := `SELECT id, given_name, family_name, gender, birth_date, phone, city, state FROM patients WHERE id = ?`
	debug.SQL(query, id)
	
	err := db.QueryRow(query, id).Scan(
		&patient.ID, &patient.GivenName, &patient.FamilyName,
		&patient.Gender, &birthDate, &phone,
		&city, &state,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query patient with ID %s: %w", id, err)
	}

	// Handle NULL values properly
	if birthDate.Valid {
		patient.BirthDate = birthDate.String
	}
	if phone.Valid {
		patient.Phone = &phone.String
	}
	if city.Valid {
		patient.City = &city.String
	}
	if state.Valid {
		patient.State = &state.String
	}

	return &patient, nil
}

func SearchPatientsByName(db *sql.DB, query string) ([]Patient, error) {
	debug.Verbose("SearchPatientsByName called with query: '%s'", query)
	query = strings.TrimSpace(query)
	
	// Extract potential name from common patterns like "patient named X", "find X", etc.
	// Common prefixes/suffixes that indicate a name follows
	namePatterns := []string{
		"patient named ",
		"patient ",
		"find ",
		"lookup ",
		"search for ",
		"named ",
		"with name ",
		"called ",
	}
	
	// Try to extract just the name part
	extractedName := query
	for _, pattern := range namePatterns {
		if strings.HasPrefix(strings.ToLower(query), strings.ToLower(pattern)) {
			extractedName = strings.TrimSpace(query[len(pattern):])
			break
		}
		if idx := strings.Index(strings.ToLower(query), strings.ToLower(" "+pattern)); idx != -1 {
			extractedName = strings.TrimSpace(query[idx+len(pattern):])
			break
		}
	}
	
	// Extract individual words from the extracted name (or original query if no pattern matched)
	// Filter out common non-name words
	commonWords := map[string]bool{
		"patient": true, "patients": true, "find": true, "search": true,
		"named": true, "called": true, "with": true, "for": true, "the": true,
		"a": true, "an": true, "lookup": true, "look": true, "up": true,
	}
	
	words := strings.Fields(extractedName)
	var wordQueries []string
	for _, word := range words {
		word = strings.TrimSpace(word)
		wordLower := strings.ToLower(word)
		// Only use words longer than 2 characters that aren't common words
		if len(word) > 2 && !commonWords[wordLower] {
			wordQueries = append(wordQueries, "%"+word+"%")
		}
	}
	
	// Also search for the extracted name as a whole if it's different from the original query
	var searchQueries []string
	if extractedName != query && len(extractedName) > 0 {
		searchQueries = append(searchQueries, "%"+extractedName+"%")
	}
	// Always search for the original query too (in case no pattern matched)
	searchQueries = append(searchQueries, "%"+query+"%")
	
	// Build the WHERE clause - search ONLY in given_name, family_name, and full name
	// We only check substrings of these name fields, nothing else
	// Ensure we have at least one search term
	if len(searchQueries) == 0 && len(wordQueries) == 0 {
		// Fallback: search for the original query if nothing was extracted
		searchQueries = []string{"%" + query + "%"}
	}
	
	whereClause := `WHERE (`
	args := []interface{}{}
	
	// Add searches for extracted name phrases
	for _, searchQuery := range searchQueries {
		if len(args) > 0 {
			whereClause += ` OR `
		}
		whereClause += `(LOWER(COALESCE(given_name, '')) LIKE LOWER(?) 
		   OR LOWER(COALESCE(family_name, '')) LIKE LOWER(?) 
		   OR LOWER(COALESCE(given_name, '') || ' ' || COALESCE(family_name, '')) LIKE LOWER(?))`
		args = append(args, searchQuery, searchQuery, searchQuery)
	}
	
	// Add individual word searches (these are the actual name parts)
	for _, wordQuery := range wordQueries {
		if len(args) > 0 {
			whereClause += ` OR `
		}
		whereClause += `(LOWER(COALESCE(given_name, '')) LIKE LOWER(?)
		   OR LOWER(COALESCE(family_name, '')) LIKE LOWER(?)
		   OR LOWER(COALESCE(given_name, '') || ' ' || COALESCE(family_name, '')) LIKE LOWER(?))`
		args = append(args, wordQuery, wordQuery, wordQuery)
	}
	
	whereClause += `)`
	
	sqlQuery := `SELECT id, given_name, family_name, gender, birth_date, phone, city, state FROM patients ` + whereClause
	debug.SQL(sqlQuery, args)
	
	rows, err := db.Query(sqlQuery, args...)

	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var patients []Patient
	for rows.Next() {
		var p Patient
		var birthDate, phone, city, state sql.NullString
		err := rows.Scan(
			&p.ID, &p.GivenName, &p.FamilyName,
			&p.Gender, &birthDate, &phone,
			&city, &state,
		)
		if err != nil {
			continue
		}

		// Handle NULL values properly
		if birthDate.Valid {
			p.BirthDate = birthDate.String
		}
		if phone.Valid {
			p.Phone = &phone.String
		}
		if city.Valid {
			p.City = &city.String
		}
		if state.Valid {
			p.State = &state.String
		}

		patients = append(patients, p)
	}

	debug.Verbose("SearchPatientsByName found %d patients", len(patients))
	return patients, nil
}

func CheckPatientExists(db *sql.DB, id string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM patients WHERE id = ?)", id).Scan(&exists)
	return exists, err
}

func UpdatePatientBirthDate(db *sql.DB, patientID, birthDate string) error {
	_, err := db.Exec("UPDATE patients SET birth_date = ? WHERE id = ?", birthDate, patientID)
	return err
}

func UpdatePatientName(db *sql.DB, patientID, givenName, familyName string) error {
	_, err := db.Exec("UPDATE patients SET given_name = ?, family_name = ? WHERE id = ?", givenName, familyName, patientID)
	return err
}

func CheckPractitionerExists(db *sql.DB, id string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM practitioners WHERE id = ?)", id).Scan(&exists)
	return exists, err
}

func GetPatientName(db *sql.DB, patientID string) (string, error) {
	var name string
	err := db.QueryRow("SELECT given_name || ' ' || family_name FROM patients WHERE id = ?", patientID).Scan(&name)
	return name, err
}

func GetEncounterStatus(db *sql.DB, encounterID string) (string, error) {
	var status string
	err := db.QueryRow("SELECT status FROM encounters WHERE id = ?", encounterID).Scan(&status)
	return status, err
}

func UpdateEncounterStatus(db *sql.DB, encounterID, status string) error {
	_, err := db.Exec("UPDATE encounters SET status = ? WHERE id = ?", status, encounterID)
	return err
}

func CreateEncounter(db *sql.DB, encounter *Encounter) error {
	_, err := db.Exec(`
		INSERT INTO encounters (
			id, resource_type, status, class, type_display,
			patient_id, practitioner_id, start_datetime
		) VALUES (?, 'Encounter', ?, ?, ?, ?, ?, ?)
	`, encounter.ID, encounter.Status, encounter.Class, encounter.TypeDisplay,
		encounter.PatientID, encounter.PractitionerID, encounter.StartDateTime)
	return err
}

func GetConditionsByPatientID(db *sql.DB, patientID string) ([]Condition, error) {
	debug.Verbose("GetConditionsByPatientID called for patient: %s", patientID)
	rows, err := db.Query(`
		SELECT id, clinical_status, code, display, patient_id, onset_datetime
		FROM conditions
		WHERE patient_id = ?
		ORDER BY onset_datetime DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conditions []Condition
	for rows.Next() {
		var c Condition
		err := rows.Scan(&c.ID, &c.ClinicalStatus, &c.Code, &c.Display, &c.PatientID, &c.OnsetDateTime)
		if err != nil {
			continue
		}
		conditions = append(conditions, c)
	}
	return conditions, nil
}

func GetMedicationsByPatientID(db *sql.DB, patientID string) ([]MedicationRequest, error) {
	debug.Verbose("GetMedicationsByPatientID called for patient: %s", patientID)
	rows, err := db.Query(`
		SELECT id, status, medication_display, patient_id, authored_on, dosage_text
		FROM medication_requests
		WHERE patient_id = ?
		ORDER BY authored_on DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var medications []MedicationRequest
	for rows.Next() {
		var m MedicationRequest
		err := rows.Scan(&m.ID, &m.Status, &m.MedicationDisplay, &m.PatientID, &m.AuthoredOn, &m.DosageText)
		if err != nil {
			continue
		}
		medications = append(medications, m)
	}
	return medications, nil
}

func GetProceduresByPatientID(db *sql.DB, patientID string) ([]Procedure, error) {
	rows, err := db.Query(`
		SELECT id, status, display, patient_id, performed_datetime
		FROM procedures
		WHERE patient_id = ?
		ORDER BY performed_datetime DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var procedures []Procedure
	for rows.Next() {
		var p Procedure
		err := rows.Scan(&p.ID, &p.Status, &p.Display, &p.PatientID, &p.PerformedDateTime)
		if err != nil {
			continue
		}
		procedures = append(procedures, p)
	}
	return procedures, nil
}

func GetImmunizationsByPatientID(db *sql.DB, patientID string) ([]Immunization, error) {
	rows, err := db.Query(`
		SELECT id, status, vaccine_display, patient_id, occurrence_datetime
		FROM immunizations
		WHERE patient_id = ?
		ORDER BY occurrence_datetime DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var immunizations []Immunization
	for rows.Next() {
		var i Immunization
		err := rows.Scan(&i.ID, &i.Status, &i.VaccineDisplay, &i.PatientID, &i.OccurrenceDateTime)
		if err != nil {
			continue
		}
		immunizations = append(immunizations, i)
	}
	return immunizations, nil
}

func GetAllergiesByPatientID(db *sql.DB, patientID string) ([]AllergyIntolerance, error) {
	rows, err := db.Query(`
		SELECT id, clinical_status, display, patient_id, criticality
		FROM allergy_intolerances
		WHERE patient_id = ?
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allergies []AllergyIntolerance
	for rows.Next() {
		var a AllergyIntolerance
		err := rows.Scan(&a.ID, &a.ClinicalStatus, &a.Display, &a.PatientID, &a.Criticality)
		if err != nil {
			continue
		}
		allergies = append(allergies, a)
	}
	return allergies, nil
}

type Medication struct {
	Code    string  `json:"code"`
	Display string  `json:"display"`
	Form    *string `json:"form,omitempty"`
}

func SearchMedicationByName(db *sql.DB, medicationName string) (*Medication, error) {
	var medication Medication

	err := db.QueryRow(`
		SELECT code, display, form
		FROM medications
		WHERE display LIKE ?
		LIMIT 1
	`, "%"+medicationName+"%").Scan(&medication.Code, &medication.Display, &medication.Form)

	if err != nil {
		return nil, err
	}

	return &medication, nil
}

func GetEncountersByPatientID(db *sql.DB, patientID string) ([]Encounter, error) {
	debug.Verbose("GetEncountersByPatientID called for patient: %s", patientID)
	rows, err := db.Query(`
		SELECT id, status, class, type_display, patient_id, practitioner_id, 
		       start_datetime, end_datetime
		FROM encounters
		WHERE patient_id = ?
		ORDER BY start_datetime DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var encounters []Encounter
	for rows.Next() {
		var e Encounter
		err := rows.Scan(&e.ID, &e.Status, &e.Class, &e.TypeDisplay, 
			&e.PatientID, &e.PractitionerID, &e.StartDateTime, &e.EndDateTime)
		if err != nil {
			continue
		}
		encounters = append(encounters, e)
	}
	return encounters, nil
}

func GetObservationsByPatientID(db *sql.DB, patientID string) ([]Observation, error) {
	debug.Verbose("GetObservationsByPatientID called for patient: %s", patientID)
	rows, err := db.Query(`
		SELECT id, status, category, code, display, patient_id, 
		       effective_datetime, value_quantity, value_unit, value_string
		FROM observations
		WHERE patient_id = ?
		ORDER BY effective_datetime DESC
	`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []Observation
	for rows.Next() {
		var o Observation
		err := rows.Scan(&o.ID, &o.Status, &o.Category, &o.Code, &o.Display,
			&o.PatientID, &o.EffectiveDateTime, &o.ValueQuantity,
			&o.ValueUnit, &o.ValueString)
		if err != nil {
			continue
		}
		observations = append(observations, o)
	}
	return observations, nil
}

func CreateObservation(db *sql.DB, observation *Observation) error {
	_, err := db.Exec(`
		INSERT INTO observations (
			id, resource_type, status, category, code, display,
			patient_id, effective_datetime, value_quantity, value_unit, value_string
		) VALUES (?, 'Observation', ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, observation.ID, observation.Status, observation.Category, observation.Code,
		observation.Display, observation.PatientID, observation.EffectiveDateTime,
		observation.ValueQuantity, observation.ValueUnit, observation.ValueString)
	return err
}
