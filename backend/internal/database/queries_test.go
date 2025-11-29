package database

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Open the actual database file in read-only mode to prevent mutations
	db, err := sql.Open("sqlite3", "file:../../database.db?mode=ro")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestSearchPatientsByName_Marty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Test searching for "Marty"
	patients, err := SearchPatientsByName(db, "Marty")
	if err != nil {
		t.Fatalf("SearchPatientsByName failed: %v", err)
	}

	// Log what we found
	t.Logf("Found %d patient(s) matching 'Marty'", len(patients))
	for i, p := range patients {
		t.Logf("Patient %d: ID=%s, Given=%s, Family=%s", i, p.ID, p.GivenName, p.FamilyName)
	}

	// Check if we found any patients
	if len(patients) == 0 {
		t.Error("No patients found with name 'Marty' - this may indicate corrupted data")
	}
}

func TestSearchPatientsByName_Various(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testQueries := []string{
		"marty",
		"MARTY", 
		"Marty",
		"Cole",
		"Smith",
		"NonexistentName999",
	}

	for _, query := range testQueries {
		t.Run(query, func(t *testing.T) {
			patients, err := SearchPatientsByName(db, query)
			if err != nil {
				t.Fatalf("SearchPatientsByName failed for '%s': %v", query, err)
			}

			t.Logf("Query '%s': found %d patients", query, len(patients))
			for i, p := range patients {
				t.Logf("  Patient %d: %s %s (ID: %s)", i, p.GivenName, p.FamilyName, p.ID)
			}
		})
	}
}


func TestGetPatientByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Test getting a known patient ID (adjust based on actual data)
	testIDs := []string{"Cole117", "Smith193", "nonexistent999"}
	
	for _, id := range testIDs {
		patient, err := GetPatientByID(db, id)
		if err != nil {
			t.Logf("GetPatientByID for '%s': %v", id, err)
		} else {
			t.Logf("Found patient: ID=%s, Name=%s %s", 
				patient.ID, patient.GivenName, patient.FamilyName)
		}
	}
}

// Debug function to directly query and log what's in the database
func TestDebugSearchQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First, let's see what's actually in the database
	rows, err := db.Query("SELECT id, given_name, family_name FROM patients")
	if err != nil {
		t.Fatalf("Failed to query patients: %v", err)
	}
	defer rows.Close()

	t.Log("Patients in database:")
	for rows.Next() {
		var id, givenName, familyName sql.NullString
		if err := rows.Scan(&id, &givenName, &familyName); err != nil {
			t.Logf("  Error scanning row: %v", err)
			continue
		}
		t.Logf("  ID: %s, Given: %s, Family: %s", 
			id.String, givenName.String, familyName.String)
	}

	// Now test the actual search
	searchQuery := "%Marty%"
	rows, err = db.Query(`
		SELECT id, given_name, family_name, gender, birth_date, phone, city, state
		FROM patients
		WHERE given_name LIKE ? OR family_name LIKE ? OR (given_name || ' ' || family_name) LIKE ?
	`, searchQuery, searchQuery, searchQuery)
	
	if err != nil {
		t.Fatalf("Search query failed: %v", err)
	}
	defer rows.Close()

	t.Log("\nSearch results for 'Marty':")
	count := 0
	for rows.Next() {
		var id string
		var givenName, familyName, gender, birthDate, phone, city, state sql.NullString
		
		err := rows.Scan(&id, &givenName, &familyName, &gender, &birthDate, &phone, &city, &state)
		if err != nil {
			t.Logf("  Error scanning result: %v", err)
			continue
		}
		
		count++
		t.Logf("  Result %d: ID=%s, Given=%s, Family=%s", 
			count, id, givenName.String, familyName.String)
	}

	if count == 0 {
		t.Error("No results found for 'Marty' - this indicates the search is failing")
	}
}
