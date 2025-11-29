package database

import (
	"database/sql"
	"fmt"
)

func GetPatientByID(db *sql.DB, id string) (*Patient, error) {
	var patient Patient
	var birthDate, phone, city, state sql.NullString

	err := db.QueryRow(`
		SELECT id, given_name, family_name, gender, birth_date, phone, city, state
		FROM patients
		WHERE id = ?
	`, id).Scan(
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
	searchQuery := "%" + query + "%"
	rows, err := db.Query(`
		SELECT id, given_name, family_name, gender, birth_date, phone, city, state
		FROM patients
		WHERE given_name LIKE ? OR family_name LIKE ? OR (given_name || ' ' || family_name) LIKE ?
	`, searchQuery, searchQuery, searchQuery)

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

	return patients, nil
}

func CheckPatientExists(db *sql.DB, id string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM patients WHERE id = ?)", id).Scan(&exists)
	return exists, err
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

func GetObservationsByPatientID(db *sql.DB, patientID string) ([]Observation, error) {
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
