package handlers

import (
	"testing"
	"time"
)

func TestCalculateAge(t *testing.T) {
	// Mock "now" by calculating relative to current time for consistent testing
	// In a real scenario, we might want to inject the "now" time into the function,
	// but for this helper function, we'll just test relative dates.
	
	now := time.Now()
	
	birthdayPassed := now.AddDate(-20, 0, -1).Format("2006-01-02")
	birthdayToday := now.AddDate(-20, 0, 0).Format("2006-01-02")
	birthdayTomorrow := now.AddDate(-20, 0, 1).Format("2006-01-02")
	
	tests := []struct {
		name      string
		birthDate string
		wantAge   int
		wantErr   bool
	}{
		{
			name:      "Valid ISO Date - Birthday Passed",
			birthDate: birthdayPassed,
			wantAge:   20,
			wantErr:   false,
		},
		{
			name:      "Valid ISO Date - Birthday Today",
			birthDate: birthdayToday,
			wantAge:   20,
			wantErr:   false,
		},
		{
			name:      "Valid ISO Date - Birthday Tomorrow",
			birthDate: birthdayTomorrow,
			wantAge:   19,
			wantErr:   false,
		},
		{
			name:      "Ambiguous Date (US style) - Should fail or be handled strictly",
			birthDate: "01/02/1990", 
			wantAge:   0,
			wantErr:   true, // We expect this to fail after our fix restricts formats
		},
		{
			name:      "Invalid Date String",
			birthDate: "not-a-date",
			wantAge:   0,
			wantErr:   true,
		},
		{
			name:      "Empty Date",
			birthDate: "",
			wantAge:   0,
			wantErr:   true,
		},
		{
			name:      "Leap Year Birthday - Feb 29",
			// Born Feb 29, 2000 (Leap Year). 
			// If today is Feb 28, 2024 (Leap Year), age should be 23.
			// If today is Feb 29, 2024, age should be 24.
			// This is hard to test deterministically without injecting "now", 
			// but we can test basic parsing.
			birthDate: "2000-02-29",
			wantAge:   now.Year() - 2000 - func() int {
				// If today is before Feb 29 (month < 2 or month == 2 and day < 29), subtract 1
				if int(now.Month()) < 2 || (int(now.Month()) == 2 && now.Day() < 29) {
					return 1
				}
				return 0
			}(),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateAge(tt.birthDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("calculateAge() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantAge {
				t.Errorf("calculateAge() = %v, want %v", got, tt.wantAge)
			}
		})
	}
}
