package model

import (
	"strings"
	"testing"
)

func validCreateRequest() CreateEventRequest {
	return CreateEventRequest{
		Title:     "Test Event",
		StartTime: "2025-06-01T10:00:00Z",
		EndTime:   "2025-06-01T11:00:00Z",
	}
}

func TestValidateRecurrenceInterval(t *testing.T) {
	r := validCreateRequest()
	r.RecurrenceFreq = "DAILY"
	r.RecurrenceInterval = -1
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "recurrence_interval must be >= 0") {
		t.Fatalf("expected interval >= 0 error, got: %v", err)
	}

	r.RecurrenceInterval = 1000
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "recurrence_interval must be at most") {
		t.Fatalf("expected interval max error, got: %v", err)
	}

	r.RecurrenceInterval = 2
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateByDay(t *testing.T) {
	tests := []struct {
		name    string
		byDay   string
		wantErr string
	}{
		{"valid simple", "MO,WE,FR", ""},
		{"valid ordinal", "2MO,-1FR", ""},
		{"invalid weekday", "MO,XX", "invalid weekday"},
		{"invalid offset", "0MO", "not zero"},
		{"too large offset", "54MO", "between -53 and 53"},
		{"bad offset format", "abMO", "invalid offset"},
		{"single char entry", "M", "invalid entry"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validCreateRequest()
			r.RecurrenceFreq = "WEEKLY"
			r.RecurrenceByDay = tt.byDay
			err := r.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateByMonthDay(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr string
	}{
		{"valid", "1,15,-1", ""},
		{"zero", "0", "not zero"},
		{"too large", "32", "between -31 and 31"},
		{"too small", "-32", "between -31 and 31"},
		{"non-number", "abc", "invalid number"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validCreateRequest()
			r.RecurrenceFreq = "MONTHLY"
			r.RecurrenceByMonthDay = tt.value
			err := r.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateByMonth(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr string
	}{
		{"valid", "1,6,12", ""},
		{"zero", "0", "between 1 and 12"},
		{"thirteen", "13", "between 1 and 12"},
		{"non-number", "abc", "invalid number"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validCreateRequest()
			r.RecurrenceFreq = "YEARLY"
			r.RecurrenceByMonth = tt.value
			err := r.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateExDates(t *testing.T) {
	r := validCreateRequest()
	r.RecurrenceFreq = "DAILY"
	r.ExDates = "2025-06-01T10:00:00Z,2025-06-02T10:00:00Z"
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r.ExDates = "not-a-date"
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "invalid RFC 3339") {
		t.Fatalf("expected RFC 3339 error, got: %v", err)
	}
}

func TestValidateRDates(t *testing.T) {
	r := validCreateRequest()
	r.RecurrenceFreq = "DAILY"
	r.RDates = "2025-06-01T10:00:00Z"
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r.RDates = "bad"
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "invalid RFC 3339") {
		t.Fatalf("expected RFC 3339 error, got: %v", err)
	}
}

func TestValidateCountUntilMutuallyExclusive(t *testing.T) {
	r := validCreateRequest()
	r.RecurrenceFreq = "DAILY"
	r.RecurrenceCount = 5
	r.RecurrenceUntil = "2025-12-31T00:00:00Z"
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually exclusive error, got: %v", err)
	}
}

func TestValidateRecurrenceFieldsWithoutFreq(t *testing.T) {
	r := validCreateRequest()
	r.RecurrenceByDay = "MO"
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "require recurrence_freq") {
		t.Fatalf("expected require freq error, got: %v", err)
	}

	r2 := validCreateRequest()
	r2.RecurrenceInterval = 2
	if err := r2.Validate(); err == nil || !strings.Contains(err.Error(), "require recurrence_freq") {
		t.Fatalf("expected require freq error, got: %v", err)
	}
}

func TestValidateUpdateRecurrenceFields(t *testing.T) {
	// RecurrenceUntil validation in update
	until := "not-a-date"
	r := &UpdateEventRequest{RecurrenceUntil: &until}
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "recurrence_until must be RFC 3339") {
		t.Fatalf("expected until format error, got: %v", err)
	}

	// RecurrenceInterval validation in update
	neg := -1
	r2 := &UpdateEventRequest{RecurrenceInterval: &neg}
	if err := r2.Validate(); err == nil || !strings.Contains(err.Error(), "recurrence_interval must be >= 0") {
		t.Fatalf("expected interval error, got: %v", err)
	}

	// ByDay validation in update
	bad := "XX"
	r3 := &UpdateEventRequest{RecurrenceByDay: &bad}
	if err := r3.Validate(); err == nil || !strings.Contains(err.Error(), "invalid weekday") {
		t.Fatalf("expected weekday error, got: %v", err)
	}

	// ByMonthDay validation in update
	badDay := "0"
	r4 := &UpdateEventRequest{RecurrenceByMonthDay: &badDay}
	if err := r4.Validate(); err == nil || !strings.Contains(err.Error(), "not zero") {
		t.Fatalf("expected monthday error, got: %v", err)
	}

	// ByMonth validation in update
	badMonth := "13"
	r5 := &UpdateEventRequest{RecurrenceByMonth: &badMonth}
	if err := r5.Validate(); err == nil || !strings.Contains(err.Error(), "between 1 and 12") {
		t.Fatalf("expected month error, got: %v", err)
	}

	// ExDates validation in update
	badEx := "not-a-date"
	r6 := &UpdateEventRequest{ExDates: &badEx}
	if err := r6.Validate(); err == nil || !strings.Contains(err.Error(), "invalid RFC 3339") {
		t.Fatalf("expected exdates error, got: %v", err)
	}

	// RDates validation in update
	badRd := "not-a-date"
	r7 := &UpdateEventRequest{RDates: &badRd}
	if err := r7.Validate(); err == nil || !strings.Contains(err.Error(), "invalid RFC 3339") {
		t.Fatalf("expected rdates error, got: %v", err)
	}
}

func TestValidateAllDayRecurrenceFields(t *testing.T) {
	r := CreateEventRequest{
		Title:          "All Day Test",
		StartTime:      "2025-06-01",
		AllDay:         true,
		RecurrenceFreq: "WEEKLY",
		RecurrenceByDay: "MO,XX",
	}
	if err := r.Validate(); err == nil || !strings.Contains(err.Error(), "invalid weekday") {
		t.Fatalf("expected weekday error for all-day event, got: %v", err)
	}
}
