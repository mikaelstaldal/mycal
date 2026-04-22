package service

import (
	"strings"
	"testing"
	"time"

	"github.com/mikaelstaldal/mycal/internal/api"
)

func validCreateReq() *api.CreateEventRequest {
	start, _ := time.Parse(time.RFC3339, "2025-06-01T10:00:00Z")
	end, _ := time.Parse(time.RFC3339, "2025-06-01T11:00:00Z")
	return &api.CreateEventRequest{
		Title:     "Test Event",
		StartTime: api.NewOptDateTime(start),
		EndTime:   api.NewOptDateTime(end),
	}
}

func TestValidateRecurrenceInterval(t *testing.T) {
	r := validCreateReq()
	r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY)
	r.RecurrenceInterval = api.NewOptInt(-1)
	if _, _, err := ValidateCreateEventRequest(r); err == nil || !strings.Contains(err.Error(), "recurrence_interval must be >= 0") {
		t.Fatalf("expected interval >= 0 error, got: %v", err)
	}

	r.RecurrenceInterval = api.NewOptInt(1000)
	if _, _, err := ValidateCreateEventRequest(r); err == nil || !strings.Contains(err.Error(), "recurrence_interval must be at most") {
		t.Fatalf("expected interval max error, got: %v", err)
	}

	r.RecurrenceInterval = api.NewOptInt(2)
	if _, _, err := ValidateCreateEventRequest(r); err != nil {
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
			r := validCreateReq()
			r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqWEEKLY)
			r.RecurrenceByDay = api.NewOptString(tt.byDay)
			_, _, err := ValidateCreateEventRequest(r)
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
			r := validCreateReq()
			r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqMONTHLY)
			r.RecurrenceByMonthday = api.NewOptString(tt.value)
			_, _, err := ValidateCreateEventRequest(r)
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
			r := validCreateReq()
			r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqYEARLY)
			r.RecurrenceByMonth = api.NewOptString(tt.value)
			_, _, err := ValidateCreateEventRequest(r)
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
	r := validCreateReq()
	r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY)
	r.Exdates = api.NewOptString("2025-06-01T10:00:00Z,2025-06-02T10:00:00Z")
	if _, _, err := ValidateCreateEventRequest(r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r.Exdates = api.NewOptString("not-a-date")
	if _, _, err := ValidateCreateEventRequest(r); err == nil || !strings.Contains(err.Error(), "invalid RFC 3339") {
		t.Fatalf("expected RFC 3339 error, got: %v", err)
	}
}

func TestValidateRDates(t *testing.T) {
	r := validCreateReq()
	r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY)
	r.Rdates = api.NewOptString("2025-06-01T10:00:00Z")
	if _, _, err := ValidateCreateEventRequest(r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r.Rdates = api.NewOptString("bad")
	if _, _, err := ValidateCreateEventRequest(r); err == nil || !strings.Contains(err.Error(), "invalid RFC 3339") {
		t.Fatalf("expected RFC 3339 error, got: %v", err)
	}
}

func TestValidateCountUntilMutuallyExclusive(t *testing.T) {
	r := validCreateReq()
	r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY)
	r.RecurrenceCount = api.NewOptInt(5)
	r.RecurrenceUntil = api.NewOptString("2025-12-31T00:00:00Z")
	if _, _, err := ValidateCreateEventRequest(r); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually exclusive error, got: %v", err)
	}
}

func TestValidateRecurrenceFieldsWithoutFreq(t *testing.T) {
	r := validCreateReq()
	r.RecurrenceByDay = api.NewOptString("MO")
	if _, _, err := ValidateCreateEventRequest(r); err == nil || !strings.Contains(err.Error(), "require recurrence_freq") {
		t.Fatalf("expected require freq error, got: %v", err)
	}

	r2 := validCreateReq()
	r2.RecurrenceInterval = api.NewOptInt(2)
	if _, _, err := ValidateCreateEventRequest(r2); err == nil || !strings.Contains(err.Error(), "require recurrence_freq") {
		t.Fatalf("expected require freq error, got: %v", err)
	}
}

func TestValidateUpdateRecurrenceFields(t *testing.T) {
	// RecurrenceUntil validation in update
	r := &api.UpdateEventRequest{RecurrenceUntil: api.NewOptString("not-a-date")}
	if err := ValidateUpdateEventRequest(r); err == nil || !strings.Contains(err.Error(), "recurrence_until must be RFC 3339") {
		t.Fatalf("expected until format error, got: %v", err)
	}

	// RecurrenceInterval validation in update
	r2 := &api.UpdateEventRequest{RecurrenceInterval: api.NewOptInt(-1)}
	if err := ValidateUpdateEventRequest(r2); err == nil || !strings.Contains(err.Error(), "recurrence_interval must be >= 0") {
		t.Fatalf("expected interval error, got: %v", err)
	}

	// ByDay validation in update
	r3 := &api.UpdateEventRequest{RecurrenceByDay: api.NewOptString("XX")}
	if err := ValidateUpdateEventRequest(r3); err == nil || !strings.Contains(err.Error(), "invalid weekday") {
		t.Fatalf("expected weekday error, got: %v", err)
	}

	// ByMonthDay validation in update
	r4 := &api.UpdateEventRequest{RecurrenceByMonthday: api.NewOptString("0")}
	if err := ValidateUpdateEventRequest(r4); err == nil || !strings.Contains(err.Error(), "not zero") {
		t.Fatalf("expected monthday error, got: %v", err)
	}

	// ByMonth validation in update
	r5 := &api.UpdateEventRequest{RecurrenceByMonth: api.NewOptString("13")}
	if err := ValidateUpdateEventRequest(r5); err == nil || !strings.Contains(err.Error(), "between 1 and 12") {
		t.Fatalf("expected month error, got: %v", err)
	}

	// ExDates validation in update
	r6 := &api.UpdateEventRequest{Exdates: api.NewOptString("not-a-date")}
	if err := ValidateUpdateEventRequest(r6); err == nil || !strings.Contains(err.Error(), "invalid RFC 3339") {
		t.Fatalf("expected exdates error, got: %v", err)
	}

	// RDates validation in update
	r7 := &api.UpdateEventRequest{Rdates: api.NewOptString("not-a-date")}
	if err := ValidateUpdateEventRequest(r7); err == nil || !strings.Contains(err.Error(), "invalid RFC 3339") {
		t.Fatalf("expected rdates error, got: %v", err)
	}
}

func TestValidateAllDayRecurrenceFields(t *testing.T) {
	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	r := &api.CreateEventRequest{
		Title:     "All Day Test",
		AllDay:    true,
		StartDate: api.NewOptDate(start),
		RecurrenceFreq: api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqWEEKLY),
		RecurrenceByDay: api.NewOptString("MO,XX"),
	}
	if _, _, err := ValidateCreateEventRequest(r); err == nil || !strings.Contains(err.Error(), "invalid weekday") {
		t.Fatalf("expected weekday error for all-day event, got: %v", err)
	}
}

func TestValidateLatitude(t *testing.T) {
	tests := []struct {
		name    string
		lat     float64
		wantErr string
	}{
		{"valid zero", 0, ""},
		{"valid positive", 45.5, ""},
		{"valid negative", -45.5, ""},
		{"valid max", 90, ""},
		{"valid min", -90, ""},
		{"too large", 90.1, "between -90 and 90"},
		{"too small", -90.1, "between -90 and 90"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validCreateReq()
			r.Latitude = api.NewOptNilFloat64(tt.lat)
			_, _, err := ValidateCreateEventRequest(r)
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

func TestValidateLongitude(t *testing.T) {
	tests := []struct {
		name    string
		lon     float64
		wantErr string
	}{
		{"valid zero", 0, ""},
		{"valid positive", 120.5, ""},
		{"valid negative", -120.5, ""},
		{"valid max", 180, ""},
		{"valid min", -180, ""},
		{"too large", 180.1, "between -180 and 180"},
		{"too small", -180.1, "between -180 and 180"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validCreateReq()
			r.Longitude = api.NewOptNilFloat64(tt.lon)
			_, _, err := ValidateCreateEventRequest(r)
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

func TestValidateUpdateCoordinates(t *testing.T) {
	r := &api.UpdateEventRequest{Latitude: api.NewOptNilFloat64(91.0)}
	if err := ValidateUpdateEventRequest(r); err == nil || !strings.Contains(err.Error(), "between -90 and 90") {
		t.Fatalf("expected latitude error, got: %v", err)
	}

	r2 := &api.UpdateEventRequest{Longitude: api.NewOptNilFloat64(181.0)}
	if err := ValidateUpdateEventRequest(r2); err == nil || !strings.Contains(err.Error(), "between -180 and 180") {
		t.Fatalf("expected longitude error, got: %v", err)
	}

	r3 := &api.UpdateEventRequest{
		Latitude:  api.NewOptNilFloat64(45.5),
		Longitude: api.NewOptNilFloat64(120.5),
	}
	if err := ValidateUpdateEventRequest(r3); err != nil {
		t.Fatalf("unexpected error for valid coordinates: %v", err)
	}
}
