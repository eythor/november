package handlers

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Pre-compiled regexes for better performance
var (
	timeRegex  = regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	slashRegex = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})(?:/(\d{2,4}))?(?:\s+(\d{1,2}):(\d{2}))?$`)
	daysRegex  = regexp.MustCompile(`in (\d+) days?`)
)

// DateOption represents a possible interpretation of an ambiguous date
type DateOption struct {
	Key         string    `json:"key"`
	DisplayText string    `json:"display_text"`
	ISODate     string    `json:"iso_date"`
	DateTime    time.Time `json:"-"`
}

// AmbiguousDateError indicates a date that has multiple valid interpretations
type AmbiguousDateError struct {
	OriginalInput string       `json:"original_input"`
	Options       []DateOption `json:"options"`
}

func (e *AmbiguousDateError) Error() string {
	return fmt.Sprintf("ambiguous date: %s", e.OriginalInput)
}

// ToUserMessage returns a user-friendly message for date confirmation
func (e *AmbiguousDateError) ToUserMessage() string {
	msg := fmt.Sprintf("The date '%s' is ambiguous. Please choose:\n\n", e.OriginalInput)
	for _, opt := range e.Options {
		msg += fmt.Sprintf("%s) %s\n", opt.Key, opt.DisplayText)
	}
	msg += "\nPlease specify which date you meant (A, B, etc.)"
	return msg
}

// Supported datetime formats
const (
	// RFC3339 and ISO 8601 variants
	FormatRFC3339           = "2006-01-02T15:04:05Z07:00"
	FormatRFC3339Nano       = "2006-01-02T15:04:05.999999999Z07:00"
	FormatISO8601           = "2006-01-02T15:04:05"
	FormatISO8601WithTZ     = "2006-01-02T15:04:05Z"
	FormatISO8601Short      = "2006-01-02T15:04"
	
	// Common datetime formats
	FormatDateTimeSpace     = "2006-01-02 15:04:05"
	FormatDateTimeSpaceShort= "2006-01-02 15:04"
	FormatDateOnly          = "2006-01-02"
	
	// German date formats
	FormatGermanDateTime    = "02.01.2006 15:04"
	FormatGermanDate        = "02.01.2006"
)

// ParseDateTimeRobust attempts to parse a datetime string in various formats
// It handles timezone conversion to Europe/Berlin and detects ambiguous dates
//
// Supported formats:
// - RFC3339: 2024-12-01T14:00:00+01:00
// - ISO 8601: 2024-12-01T14:00:05
// - Date + Time: 2024-12-01 14:00
// - German format: 01.12.2024 14:00
// - Natural language: tomorrow, next Monday, etc.
// - Date only: 2024-12-01 (defaults to 09:00)
//
// Returns:
// - Parsed time in Europe/Berlin timezone
// - AmbiguousDateError if multiple interpretations exist
// - Standard error for invalid dates
func ParseDateTimeRobust(input string, referenceTime time.Time) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, fmt.Errorf("empty datetime string")
	}
	
	// Load Berlin timezone for consistent handling
	berlinTZ, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		// Fallback to UTC if timezone loading fails
		berlinTZ = time.UTC
	}
	
	// Try parsing in order of specificity
	
	// 1. RFC3339 with timezone (most specific)
	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t.In(berlinTZ), nil
	}
	
	if t, err := time.Parse(FormatRFC3339Nano, input); err == nil {
		return t.In(berlinTZ), nil
	}
	
	// 2. ISO 8601 variants
	formats := []string{
		FormatISO8601WithTZ,
		FormatISO8601,
		FormatISO8601Short,
		FormatDateTimeSpace,
		FormatDateTimeSpaceShort,
	}
	
	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, berlinTZ); err == nil {
			return t, nil
		}
	}
	
	// 3. Date only (default to 09:00 Berlin time)
	if t, err := time.ParseInLocation(FormatDateOnly, input, berlinTZ); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, berlinTZ), nil
	}
	
	// 4. German format
	if t, err := time.ParseInLocation(FormatGermanDateTime, input, berlinTZ); err == nil {
		return t, nil
	}
	
	if t, err := time.ParseInLocation(FormatGermanDate, input, berlinTZ); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, berlinTZ), nil
	}
	
	// 5. Handle potentially ambiguous slash formats (MM/DD vs DD/MM)
	if slashDate, err := parseSlashFormat(input, berlinTZ, referenceTime); err == nil {
		return slashDate, nil
	} else if ambigErr, ok := err.(*AmbiguousDateError); ok {
		return time.Time{}, ambigErr
	}
	
	// 6. Try natural language parsing
	if t, err := parseRelativeDate(input, referenceTime, berlinTZ); err == nil {
		return t, nil
	}
	
	// All parsing attempts failed
	return time.Time{}, fmt.Errorf("unable to parse datetime '%s'. Supported formats: RFC3339 (2024-12-01T14:00:00+01:00), ISO 8601 (2024-12-01T14:00:00), Date+Time (2024-12-01 14:00), German (01.12.2024 14:00), natural language (tomorrow at 14:00), or Date only (2024-12-01)", input)
}

// parseSlashFormat handles MM/DD and DD/MM ambiguity
func parseSlashFormat(input string, tz *time.Location, ref time.Time) (time.Time, error) {
	matches := slashRegex.FindStringSubmatch(input)
	
	if matches == nil {
		return time.Time{}, fmt.Errorf("not a slash format")
	}
	
	var first, second, year, hour, minute int
	fmt.Sscanf(matches[1], "%d", &first)
	fmt.Sscanf(matches[2], "%d", &second)
	
	// Default year to current year if not specified
	year = ref.Year()
	if matches[3] != "" {
		fmt.Sscanf(matches[3], "%d", &year)
		if year < 100 {
			year += 2000
		}
	}
	
	// Parse time component or default to 09:00
	hour = 9
	minute = 0
	if matches[4] != "" {
		fmt.Sscanf(matches[4], "%d", &hour)
		fmt.Sscanf(matches[5], "%d", &minute)
	}
	
	// Validate ranges
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return time.Time{}, fmt.Errorf("invalid time component: %02d:%02d", hour, minute)
	}
	
	// Check if both interpretations are valid (ambiguous)
	firstIsValidMonth := first >= 1 && first <= 12
	secondIsValidMonth := second >= 1 && second <= 12
	firstIsValidDay := first >= 1 && first <= 31
	secondIsValidDay := second >= 1 && second <= 31
	
	// If both could be months (and days), it's ambiguous
	if firstIsValidMonth && secondIsValidMonth && firstIsValidDay && secondIsValidDay && first != second {
		optionMM := time.Date(year, time.Month(first), second, hour, minute, 0, 0, tz)
		optionDD := time.Date(year, time.Month(second), first, hour, minute, 0, 0, tz)
		
		return time.Time{}, &AmbiguousDateError{
			OriginalInput: input,
			Options: []DateOption{
				{
					Key:         "A",
					DisplayText: fmt.Sprintf("MM/DD format: %s", optionMM.Format("Monday, January 2, 2006 at 15:04")),
					ISODate:     optionMM.Format(time.RFC3339),
					DateTime:    optionMM,
				},
				{
					Key:         "B",
					DisplayText: fmt.Sprintf("DD/MM format: %s", optionDD.Format("Monday, January 2, 2006 at 15:04")),
					ISODate:     optionDD.Format(time.RFC3339),
					DateTime:    optionDD,
				},
			},
		}
	}
	
	// Only one valid interpretation - prefer DD/MM (European)
	if secondIsValidMonth && firstIsValidDay {
		return time.Date(year, time.Month(second), first, hour, minute, 0, 0, tz), nil
	}
	
	// Try MM/DD (US format)
	if firstIsValidMonth && secondIsValidDay {
		return time.Date(year, time.Month(first), second, hour, minute, 0, 0, tz), nil
	}
	
	return time.Time{}, fmt.Errorf("invalid date components: %d/%d", first, second)
}

// parseRelativeDate handles natural language expressions like "tomorrow", "next week"
func parseRelativeDate(input string, ref time.Time, tz *time.Location) (time.Time, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	now := ref.In(tz)
	
	// Extract time if present
	hour, minute := 9, 0 // Default to 09:00
	
	if matches := timeRegex.FindStringSubmatch(input); matches != nil {
		fmt.Sscanf(matches[1], "%d", &hour)
		fmt.Sscanf(matches[2], "%d", &minute)
	} else {
		// Check for common time expressions
		if strings.Contains(input, "morning") {
			hour = 9
		} else if strings.Contains(input, "noon") || strings.Contains(input, "midday") {
			hour = 12
		} else if strings.Contains(input, "afternoon") {
			hour = 14
		} else if strings.Contains(input, "evening") {
			hour = 18
		}
	}
	
	// Relative day expressions
	switch {
	case strings.Contains(input, "today"):
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, tz), nil
	
	case strings.Contains(input, "tomorrow"):
		return time.Date(now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, tz), nil
	
	case strings.Contains(input, "next week"):
		return time.Date(now.Year(), now.Month(), now.Day()+7, hour, minute, 0, 0, tz), nil
	
	case daysRegex.MatchString(input):
		matches := daysRegex.FindStringSubmatch(input)
		var days int
		fmt.Sscanf(matches[1], "%d", &days)
		return time.Date(now.Year(), now.Month(), now.Day()+days, hour, minute, 0, 0, tz), nil
	
	case strings.Contains(input, "next monday"):
		return nextWeekday(now, time.Monday, hour, minute, tz), nil
	case strings.Contains(input, "next tuesday"):
		return nextWeekday(now, time.Tuesday, hour, minute, tz), nil
	case strings.Contains(input, "next wednesday"):
		return nextWeekday(now, time.Wednesday, hour, minute, tz), nil
	case strings.Contains(input, "next thursday"):
		return nextWeekday(now, time.Thursday, hour, minute, tz), nil
	case strings.Contains(input, "next friday"):
		return nextWeekday(now, time.Friday, hour, minute, tz), nil
	case strings.Contains(input, "next saturday"):
		return nextWeekday(now, time.Saturday, hour, minute, tz), nil
	case strings.Contains(input, "next sunday"):
		return nextWeekday(now, time.Sunday, hour, minute, tz), nil
	}
	
	return time.Time{}, fmt.Errorf("unrecognized relative date expression")
}

// nextWeekday calculates the next occurrence of a specific weekday
func nextWeekday(ref time.Time, targetDay time.Weekday, hour, minute int, tz *time.Location) time.Time {
	daysUntil := int(targetDay) - int(ref.Weekday())
	if daysUntil <= 0 {
		daysUntil += 7 // Next week
	}
	return time.Date(ref.Year(), ref.Month(), ref.Day()+daysUntil, hour, minute, 0, 0, tz)
}

// ValidateDateTime performs validation on a parsed datetime
func ValidateDateTime(t time.Time) error {
	// Check if date is in the past (allow up to 24 hours for flexibility)
	now := time.Now()
	if t.Before(now.Add(-24 * time.Hour)) {
		return fmt.Errorf("datetime is in the past: %s", t.Format("2006-01-02 15:04"))
	}
	
	// Check if date is too far in the future (e.g., more than 2 years)
	twoYearsFromNow := now.AddDate(2, 0, 0)
	if t.After(twoYearsFromNow) {
		return fmt.Errorf("datetime is too far in the future: %s (max 2 years ahead)", t.Format("2006-01-02 15:04"))
	}
	
	return nil
}

// FormatDateTimeForDisplay formats a datetime in a readable format
func FormatDateTimeForDisplay(t time.Time) string {
	berlinTZ, _ := time.LoadLocation("Europe/Berlin")
	localTime := t.In(berlinTZ)
	return localTime.Format("Monday, January 2, 2006 at 15:04 MST")
}