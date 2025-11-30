package handlers

import (
	"strings"
	"testing"
	"time"
)

func TestParseDateTimeRobust(t *testing.T) {
	// Use a fixed reference time for consistent testing
	refTime := time.Date(2024, 11, 30, 10, 0, 0, 0, time.UTC)
	berlinTZ, _ := time.LoadLocation("Europe/Berlin")
	
	tests := []struct {
		name        string
		input       string
		wantYear    int
		wantMonth   time.Month
		wantDay     int
		wantHour    int
		wantMinute  int
		wantErr     bool
		wantAmbig   bool
	}{
		// RFC3339 format
		{
			name:       "RFC3339 with timezone",
			input:      "2024-12-01T14:30:00+01:00",
			wantYear:   2024,
			wantMonth:  12,
			wantDay:    1,
			wantHour:   14,
			wantMinute: 30,
		},
		
		// ISO 8601 format
		{
			name:       "ISO 8601 without timezone",
			input:      "2024-12-15T09:00:00",
			wantYear:   2024,
			wantMonth:  12,
			wantDay:    15,
			wantHour:   9,
			wantMinute: 0,
		},
		
		// Date with space and time
		{
			name:       "Date space time",
			input:      "2024-12-25 18:30",
			wantYear:   2024,
			wantMonth:  12,
			wantDay:    25,
			wantHour:   18,
			wantMinute: 30,
		},
		
		// Date only (defaults to 09:00)
		{
			name:       "Date only defaults to 09:00",
			input:      "2024-12-01",
			wantYear:   2024,
			wantMonth:  12,
			wantDay:    1,
			wantHour:   9,
			wantMinute: 0,
		},
		
		// German format
		{
			name:       "German date format with time",
			input:      "15.12.2024 14:30",
			wantYear:   2024,
			wantMonth:  12,
			wantDay:    15,
			wantHour:   14,
			wantMinute: 30,
		},
		{
			name:       "German date only",
			input:      "25.12.2024",
			wantYear:   2024,
			wantMonth:  12,
			wantDay:    25,
			wantHour:   9,
			wantMinute: 0,
		},
		
		// Slash format - unambiguous
		{
			name:       "Unambiguous DD/MM format",
			input:      "25/12/2024",
			wantYear:   2024,
			wantMonth:  12,
			wantDay:    25,
			wantHour:   9,
			wantMinute: 0,
		},
		
		// Ambiguous format (should return error)
		{
			name:      "Ambiguous MM/DD vs DD/MM",
			input:     "06/12/2024",
			wantAmbig: true,
			wantErr:   true,
		},
		
		// Invalid inputs
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Invalid format",
			input:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "Invalid date components",
			input:   "99/99/2024",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDateTimeRobust(tt.input, refTime)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDateTimeRobust() expected error, got nil")
				}
				if tt.wantAmbig {
					if _, ok := err.(*AmbiguousDateError); !ok {
						t.Errorf("ParseDateTimeRobust() expected AmbiguousDateError, got %T", err)
					}
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseDateTimeRobust() unexpected error = %v", err)
				return
			}
			
			// Convert to Berlin timezone for comparison
			gotBerlin := got.In(berlinTZ)
			
			if gotBerlin.Year() != tt.wantYear {
				t.Errorf("Year = %v, want %v", gotBerlin.Year(), tt.wantYear)
			}
			if gotBerlin.Month() != tt.wantMonth {
				t.Errorf("Month = %v, want %v", gotBerlin.Month(), tt.wantMonth)
			}
			if gotBerlin.Day() != tt.wantDay {
				t.Errorf("Day = %v, want %v", gotBerlin.Day(), tt.wantDay)
			}
			if gotBerlin.Hour() != tt.wantHour {
				t.Errorf("Hour = %v, want %v", gotBerlin.Hour(), tt.wantHour)
			}
			if gotBerlin.Minute() != tt.wantMinute {
				t.Errorf("Minute = %v, want %v", gotBerlin.Minute(), tt.wantMinute)
			}
		})
	}
}

func TestParseRelativeDate(t *testing.T) {
	// Fixed reference: Saturday, November 30, 2024, 10:00
	refTime := time.Date(2024, 11, 30, 10, 0, 0, 0, time.UTC)
	berlinTZ, _ := time.LoadLocation("Europe/Berlin")
	
	tests := []struct {
		name       string
		input      string
		wantDay    int
		wantMonth  time.Month
		wantHour   int
		wantErr    bool
	}{
		{
			name:      "tomorrow",
			input:     "tomorrow",
			wantDay:   1, // December 1st
			wantMonth: 12,
			wantHour:  9,
		},
		{
			name:      "tomorrow at 14:30",
			input:     "tomorrow at 14:30",
			wantDay:   1,
			wantMonth: 12,
			wantHour:  14,
		},
		{
			name:      "next Monday",
			input:     "next monday",
			wantDay:   2, // December 2nd (Monday)
			wantMonth: 12,
			wantHour:  9,
		},
		{
			name:      "in 3 days",
			input:     "in 3 days",
			wantDay:   3, // December 3rd
			wantMonth: 12,
			wantHour:  9,
		},
		{
			name:    "unrecognized expression",
			input:   "some random text",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRelativeDate(tt.input, refTime, berlinTZ)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseRelativeDate() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("parseRelativeDate() unexpected error = %v", err)
				return
			}
			
			if got.Day() != tt.wantDay {
				t.Errorf("Day = %v, want %v", got.Day(), tt.wantDay)
			}
			if got.Month() != tt.wantMonth {
				t.Errorf("Month = %v, want %v", got.Month(), tt.wantMonth)
			}
			if got.Hour() != tt.wantHour {
				t.Errorf("Hour = %v, want %v", got.Hour(), tt.wantHour)
			}
		})
	}
}

func TestValidateDateTime(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name    string
		input   time.Time
		wantErr bool
	}{
		{
			name:    "future date is valid",
			input:   now.AddDate(0, 1, 0), // 1 month in future
			wantErr: false,
		},
		{
			name:    "past date is invalid",
			input:   now.AddDate(0, 0, -2), // 2 days ago
			wantErr: true,
		},
		{
			name:    "too far future is invalid",
			input:   now.AddDate(3, 0, 0), // 3 years in future
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDateTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDateTime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAmbiguousDateError(t *testing.T) {
	ambigErr := &AmbiguousDateError{
		OriginalInput: "06/12/2024",
		Options: []DateOption{
			{Key: "A", DisplayText: "June 12, 2024"},
			{Key: "B", DisplayText: "December 6, 2024"},
		},
	}
	
	msg := ambigErr.ToUserMessage()
	
	if !strings.Contains(msg, "06/12/2024") {
		t.Errorf("Message should contain original input")
	}
	if !strings.Contains(msg, "A)") || !strings.Contains(msg, "B)") {
		t.Errorf("Message should contain both options")
	}
}