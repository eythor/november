package database

import (
	"database/sql"
	"fmt"
	
	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

type Patient struct {
	ID         string  `json:"id"`
	GivenName  string  `json:"given_name"`
	FamilyName string  `json:"family_name"`
	Gender     string  `json:"gender"`
	BirthDate  string  `json:"birth_date"`
	Phone      *string `json:"phone,omitempty"`
	City       *string `json:"city,omitempty"`
	State      *string `json:"state,omitempty"`
}

type Encounter struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	Class          string  `json:"class"`
	TypeDisplay    *string `json:"type_display,omitempty"`
	PatientID      string  `json:"patient_id"`
	PractitionerID *string `json:"practitioner_id,omitempty"`
	StartDateTime  string  `json:"start_datetime"`
	EndDateTime    *string `json:"end_datetime,omitempty"`
}

type Condition struct {
	ID              string  `json:"id"`
	ClinicalStatus  string  `json:"clinical_status"`
	Code            string  `json:"code"`
	Display         string  `json:"display"`
	PatientID       string  `json:"patient_id"`
	OnsetDateTime   *string `json:"onset_datetime,omitempty"`
}

type MedicationRequest struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	MedicationDisplay string  `json:"medication_display"`
	PatientID         string  `json:"patient_id"`
	AuthoredOn        string  `json:"authored_on"`
	DosageText        *string `json:"dosage_text,omitempty"`
}

type Procedure struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	Display           string  `json:"display"`
	PatientID         string  `json:"patient_id"`
	PerformedDateTime *string `json:"performed_datetime,omitempty"`
}

type Immunization struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	VaccineDisplay  string `json:"vaccine_display"`
	PatientID       string `json:"patient_id"`
	OccurrenceDateTime string `json:"occurrence_datetime"`
}

type AllergyIntolerance struct {
	ID              string  `json:"id"`
	ClinicalStatus  string  `json:"clinical_status"`
	Display         string  `json:"display"`
	PatientID       string  `json:"patient_id"`
	Criticality     *string `json:"criticality,omitempty"`
}

type Observation struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	Category          string  `json:"category"`
	Code              string  `json:"code"`
	Display           string  `json:"display"`
	PatientID         string  `json:"patient_id"`
	EffectiveDateTime string  `json:"effective_datetime"`
	ValueQuantity     *float64 `json:"value_quantity,omitempty"`
	ValueUnit         *string  `json:"value_unit,omitempty"`
	ValueString       *string  `json:"value_string,omitempty"`
}