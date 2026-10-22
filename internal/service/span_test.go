package service

import (
	"testing"
	"time"

	"github.com/mikaelstaldal/mycal/internal/api"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinalEventSpanWritePaths(t *testing.T) {
	parse := func(s string) time.Time { v, err := time.Parse(time.RFC3339, s); require.NoError(t, err); return v }
	for _, tc := range []struct {
		name  string
		req   api.UpdateEventRequest
		valid bool
	}{
		{name: "end year", req: api.UpdateEventRequest{EndTime: api.NewOptDateTime(parse("9999-12-31T00:00:00Z"))}},
		{name: "start year", req: api.UpdateEventRequest{StartTime: api.NewOptDateTime(parse("1500-01-01T00:00:00Z"))}},
		{name: "merged duration", req: api.UpdateEventRequest{EndTime: api.NewOptDateTime(parse("2028-01-01T00:00:00Z"))}},
		{name: "duration", req: api.UpdateEventRequest{Duration: api.NewOptString("P367D")}},
		{name: "backwards", req: api.UpdateEventRequest{EndTime: api.NewOptDateTime(parse("2000-01-01T00:00:00Z"))}},
		{name: "equal", req: api.UpdateEventRequest{EndTime: api.NewOptDateTime(parse("2026-01-01T00:00:00Z"))}},
		{name: "new start and duration", req: api.UpdateEventRequest{StartTime: api.NewOptDateTime(parse("2028-01-01T00:00:00Z")), Duration: api.NewOptString("PT1H")}, valid: true},
		{name: "boundary", req: api.UpdateEventRequest{Duration: api.NewOptString("P366D")}, valid: true},
	} {
		for _, path := range []string{"update", "new override", "existing override"} {
			t.Run(tc.name+" "+path, func(t *testing.T) {
				written := false
				repo := &mockRepo{
					getByIDFn: func(int64) (*model.Event, error) {
						return &model.Event{ID: 1, Title: "event", StartTime: "2026-01-01T00:00:00Z", EndTime: "2026-01-01T01:00:00Z", RecurrenceFreq: "DAILY"}, nil
					},
					createFn: func(*model.Event) error { written = true; return nil },
					updateFn: func(*model.Event) error { written = true; return nil },
				}
				if path == "existing override" {
					repo.getOverrideFn = func(int64, string) (*model.Event, error) { return repo.getByIDFn(1) }
				}
				svc := NewEventService(repo, nil)
				var err error
				if path != "update" {
					_, err = svc.CreateOrUpdateOverride(1, "2026-01-01T00:00:00Z", &tc.req)
				} else {
					_, err = svc.Update(1, &tc.req)
				}
				if tc.valid {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, ErrValidation)
				}
				assert.Equal(t, tc.valid, written)
			})
		}
	}
}

func TestCreateFinalDurationSpan(t *testing.T) {
	for _, allDay := range []bool{false, true} {
		for _, duration := range []string{"P366D", "P367D", "P36500D"} {
			req := &api.CreateEventRequest{Title: "event", AllDay: allDay, Duration: api.NewOptString(duration)}
			start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			if allDay {
				req.StartDate = api.NewOptDate(start)
			} else {
				req.StartTime = api.NewOptDateTime(start)
			}
			_, _, err := ValidateCreateEventRequest(req)
			if duration == "P366D" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		}
		start := time.Date(time.Now().Year()+maxYearOffset, 12, 31, 0, 0, 0, 0, time.UTC)
		req := &api.CreateEventRequest{Title: "boundary", AllDay: allDay, Duration: api.NewOptString("P1D")}
		if allDay {
			req.StartDate = api.NewOptDate(start)
		} else {
			req.StartTime = api.NewOptDateTime(start)
		}
		_, _, err := ValidateCreateEventRequest(req)
		require.ErrorContains(t, err, "date must be between year")
	}
}
