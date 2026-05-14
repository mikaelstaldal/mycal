package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
	_, _, err := ValidateCreateEventRequest(r)
	assert.ErrorContains(t, err, "recurrence_interval must be >= 0")

	r.RecurrenceInterval = api.NewOptInt(1000)
	_, _, err = ValidateCreateEventRequest(r)
	assert.ErrorContains(t, err, "recurrence_interval must be at most")

	r.RecurrenceInterval = api.NewOptInt(2)
	_, _, err = ValidateCreateEventRequest(r)
	assert.NoError(t, err)
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
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
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
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
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
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateExDates(t *testing.T) {
	r := validCreateReq()
	r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY)
	r.Exdates = api.NewOptString("2025-06-01T10:00:00Z,2025-06-02T10:00:00Z")
	_, _, err := ValidateCreateEventRequest(r)
	assert.NoError(t, err)

	r.Exdates = api.NewOptString("not-a-date")
	_, _, err = ValidateCreateEventRequest(r)
	assert.ErrorContains(t, err, "invalid RFC 3339")
}

func TestValidateRDates(t *testing.T) {
	r := validCreateReq()
	r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY)
	r.Rdates = api.NewOptString("2025-06-01T10:00:00Z")
	_, _, err := ValidateCreateEventRequest(r)
	assert.NoError(t, err)

	r.Rdates = api.NewOptString("bad")
	_, _, err = ValidateCreateEventRequest(r)
	assert.ErrorContains(t, err, "invalid RFC 3339")
}

func TestValidateCountUntilMutuallyExclusive(t *testing.T) {
	r := validCreateReq()
	r.RecurrenceFreq = api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqDAILY)
	r.RecurrenceCount = api.NewOptInt(5)
	r.RecurrenceUntil = api.NewOptString("2025-12-31T00:00:00Z")
	_, _, err := ValidateCreateEventRequest(r)
	assert.ErrorContains(t, err, "mutually exclusive")
}

func TestValidateRecurrenceFieldsWithoutFreq(t *testing.T) {
	r := validCreateReq()
	r.RecurrenceByDay = api.NewOptString("MO")
	_, _, err := ValidateCreateEventRequest(r)
	assert.ErrorContains(t, err, "require recurrence_freq")

	r2 := validCreateReq()
	r2.RecurrenceInterval = api.NewOptInt(2)
	_, _, err = ValidateCreateEventRequest(r2)
	assert.ErrorContains(t, err, "require recurrence_freq")
}

func TestValidateUpdateRecurrenceFields(t *testing.T) {
	// RecurrenceUntil validation in update
	r := &api.UpdateEventRequest{RecurrenceUntil: api.NewOptString("not-a-date")}
	assert.ErrorContains(t, ValidateUpdateEventRequest(r), "recurrence_until must be RFC 3339")

	// RecurrenceInterval validation in update
	r2 := &api.UpdateEventRequest{RecurrenceInterval: api.NewOptInt(-1)}
	assert.ErrorContains(t, ValidateUpdateEventRequest(r2), "recurrence_interval must be >= 0")

	// ByDay validation in update
	r3 := &api.UpdateEventRequest{RecurrenceByDay: api.NewOptString("XX")}
	assert.ErrorContains(t, ValidateUpdateEventRequest(r3), "invalid weekday")

	// ByMonthDay validation in update
	r4 := &api.UpdateEventRequest{RecurrenceByMonthday: api.NewOptString("0")}
	assert.ErrorContains(t, ValidateUpdateEventRequest(r4), "not zero")

	// ByMonth validation in update
	r5 := &api.UpdateEventRequest{RecurrenceByMonth: api.NewOptString("13")}
	assert.ErrorContains(t, ValidateUpdateEventRequest(r5), "between 1 and 12")

	// ExDates validation in update
	r6 := &api.UpdateEventRequest{Exdates: api.NewOptString("not-a-date")}
	assert.ErrorContains(t, ValidateUpdateEventRequest(r6), "invalid RFC 3339")

	// RDates validation in update
	r7 := &api.UpdateEventRequest{Rdates: api.NewOptString("not-a-date")}
	assert.ErrorContains(t, ValidateUpdateEventRequest(r7), "invalid RFC 3339")
}

func TestValidateAllDayRecurrenceFields(t *testing.T) {
	start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	r := &api.CreateEventRequest{
		Title:           "All Day Test",
		AllDay:          true,
		StartDate:       api.NewOptDate(start),
		RecurrenceFreq:  api.NewOptCreateEventRequestRecurrenceFreq(api.CreateEventRequestRecurrenceFreqWEEKLY),
		RecurrenceByDay: api.NewOptString("MO,XX"),
	}
	_, _, err := ValidateCreateEventRequest(r)
	assert.ErrorContains(t, err, "invalid weekday")
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
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
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
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateCoordinates(t *testing.T) {
	r := &api.UpdateEventRequest{Latitude: api.NewOptNilFloat64(91.0)}
	assert.ErrorContains(t, ValidateUpdateEventRequest(r), "between -90 and 90")

	r2 := &api.UpdateEventRequest{Longitude: api.NewOptNilFloat64(181.0)}
	assert.ErrorContains(t, ValidateUpdateEventRequest(r2), "between -180 and 180")

	r3 := &api.UpdateEventRequest{
		Latitude:  api.NewOptNilFloat64(45.5),
		Longitude: api.NewOptNilFloat64(120.5),
	}
	assert.NoError(t, ValidateUpdateEventRequest(r3))
}
