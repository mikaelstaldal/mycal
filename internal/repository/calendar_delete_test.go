package repository

import (
	"database/sql"
	"testing"

	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCalendar(t *testing.T) {
	r := newTestRepo(t)
	cal := &model.Calendar{Name: "Delete me", Color: "red"}
	require.NoError(t, r.CreateCalendar(cal))
	feed := &model.Feed{URL: "https://example.com/calendar.ics", CalendarID: cal.ID}
	require.NoError(t, r.CreateFeed(feed))
	parent := &model.Event{Title: "Series", StartTime: "2026-10-21T10:00:00Z", EndTime: "2026-10-21T11:00:00Z", CalendarID: cal.ID, RecurrenceFreq: "DAILY"}
	require.NoError(t, r.Create(parent))
	override := &model.Event{Title: "Override", StartTime: parent.StartTime, EndTime: parent.EndTime, CalendarID: cal.ID, RecurrenceParentID: &parent.ID}
	require.NoError(t, r.Create(override))
	other := &model.Event{Title: "Keep", StartTime: parent.StartTime, EndTime: parent.EndTime}
	require.NoError(t, r.Create(other))
	require.NoError(t, r.DeleteCalendar(cal.ID))
	got, err := r.GetCalendarByID(cal.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
	events, err := r.ListAll(nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, other.ID, events[0].ID)
	feeds, err := r.ListFeeds()
	require.NoError(t, err)
	assert.Empty(t, feeds)
	assert.ErrorIs(t, r.DeleteCalendar(cal.ID), sql.ErrNoRows)
	require.Error(t, r.DeleteCalendar(0))
	got, err = r.GetCalendarByID(0)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestDeleteCalendarRollback(t *testing.T) {
	r := newTestRepo(t)
	cal := &model.Calendar{Name: "Keep on failure", Color: "red"}
	require.NoError(t, r.CreateCalendar(cal))
	_, err := r.db.Exec(`CREATE TRIGGER fail_calendar_events BEFORE DELETE ON events BEGIN SELECT RAISE(ABORT, 'failure'); END`)
	require.NoError(t, err)
	e := &model.Event{Title: "Keep", StartTime: "2026-10-21T10:00:00Z", EndTime: "2026-10-21T11:00:00Z", CalendarID: cal.ID}
	require.NoError(t, r.Create(e))
	require.Error(t, r.DeleteCalendar(cal.ID))
	got, err := r.GetCalendarByID(cal.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	events, err := r.ListAll(nil)
	require.NoError(t, err)
	assert.Len(t, events, 1)
}
