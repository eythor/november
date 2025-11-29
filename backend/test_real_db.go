package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/eythor/mcp-server/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./database.db"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Printf("Testing database at: %s\n\n", dbPath)

	// First, let's see what's in the patients table
	fmt.Println("=== All patients in database ===")
	rows, err := db.Query("SELECT id, given_name, family_name FROM patients LIMIT 10")
	if err != nil {
		log.Fatalf("Failed to query patients: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id sql.NullString
		var givenName, familyName sql.NullString
		if err := rows.Scan(&id, &givenName, &familyName); err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			continue
		}
		count++
		fmt.Printf("%d. ID: %s, Given: %s, Family: %s\n", 
			count, id.String, givenName.String, familyName.String)
	}
	fmt.Printf("Total patients shown: %d\n\n", count)

	// Now test searching for Marty
	fmt.Println("=== Testing SearchPatientsByName for 'Marty' ===")
	patients, err := database.SearchPatientsByName(db, "Marty")
	if err != nil {
		fmt.Printf("ERROR: SearchPatientsByName failed: %v\n", err)
	} else {
		fmt.Printf("Found %d patients matching 'Marty'\n", len(patients))
		for i, p := range patients {
			fmt.Printf("%d. ID=%s, Given=%s, Family=%s\n", 
				i+1, p.ID, p.GivenName, p.FamilyName)
		}
	}

	// Test raw SQL query for Marty
	fmt.Println("\n=== Raw SQL search for 'Marty' ===")
	searchQuery := "%Marty%"
	rows2, err := db.Query(`
		SELECT id, given_name, family_name 
		FROM patients
		WHERE given_name LIKE ? OR family_name LIKE ? OR (given_name || ' ' || family_name) LIKE ?
		LIMIT 10
	`, searchQuery, searchQuery, searchQuery)
	
	if err != nil {
		fmt.Printf("ERROR: Raw query failed: %v\n", err)
	} else {
		defer rows2.Close()
		count = 0
		for rows2.Next() {
			var id string
			var givenName, familyName sql.NullString
			
			err := rows2.Scan(&id, &givenName, &familyName)
			if err != nil {
				fmt.Printf("Error scanning result: %v\n", err)
				continue
			}
			
			count++
			fmt.Printf("%d. ID=%s, Given=%s, Family=%s\n", 
				count, id, givenName.String, familyName.String)
		}
		fmt.Printf("Total found: %d\n", count)
	}

	// Check for any patients with 'Marty' anywhere in their data
	fmt.Println("\n=== Case-insensitive search for 'marty' ===")
	rows3, err := db.Query(`
		SELECT id, given_name, family_name 
		FROM patients
		WHERE LOWER(given_name) LIKE LOWER(?) 
		   OR LOWER(family_name) LIKE LOWER(?) 
		   OR LOWER(given_name || ' ' || family_name) LIKE LOWER(?)
		LIMIT 10
	`, "%marty%", "%marty%", "%marty%")
	
	if err != nil {
		fmt.Printf("ERROR: Case-insensitive query failed: %v\n", err)
	} else {
		defer rows3.Close()
		count = 0
		for rows3.Next() {
			var id string
			var givenName, familyName sql.NullString
			
			err := rows3.Scan(&id, &givenName, &familyName)
			if err != nil {
				fmt.Printf("Error scanning result: %v\n", err)
				continue
			}
			
			count++
			fmt.Printf("%d. ID=%s, Given=%s, Family=%s\n", 
				count, id, givenName.String, familyName.String)
		}
		fmt.Printf("Total found: %d\n", count)
	}
}