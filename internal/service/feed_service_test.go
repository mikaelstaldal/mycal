package service

import (
	"fmt"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedFetchURL(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
		fail bool
	}{
		{"webcal", "webcal://1.1.1.1/calendar.ics?token=a%2Fb", "https://1.1.1.1/calendar.ics?token=a%2Fb", false},
		{"webcals with port", "webcals://1.1.1.1:8443/calendar.ics", "https://1.1.1.1:8443/calendar.ics", false},
		{"uppercase scheme", "WEBCAL://1.1.1.1/calendar.ics", "https://1.1.1.1/calendar.ics", false},
		{"https", "https://1.1.1.1/calendar.ics", "https://1.1.1.1/calendar.ics", false},
		{"http", "http://1.1.1.1/calendar.ics", "http://1.1.1.1/calendar.ics", false},
		{"private address", "webcal://127.0.0.1/calendar.ics", "", true},
		{"other scheme", "file:///etc/passwd", "", true},
		{"missing host", "webcal:calendar.ics", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := feedFetchURL(tt.uri)
			if tt.fail {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFeedSnapshotReplacement(t *testing.T) {
	db, err := repository.OpenDB(repository.MemoryDSN(t.Name()), 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo, err := repository.NewSQLiteRepository(db)
	require.NoError(t, err)
	service := NewFeedService(repo, repo, repo)
	feed := &model.Feed{URL: "https://example.com/feed", RefreshIntervalMinutes: 60}
	require.NoError(t, repo.CreateFeed(feed))
	other := &model.Feed{URL: "https://example.com/other", RefreshIntervalMinutes: 60}
	require.NoError(t, repo.CreateFeed(other))
	manual := &model.Event{Title: "Manual", StartTime: "2026-10-20T10:00:00Z", EndTime: "2026-10-20T11:00:00Z"}
	require.NoError(t, repo.Create(manual))
	event := func(uid, title string) string {
		return "BEGIN:VEVENT\r\n" + uid + "SUMMARY:" + title + "\r\nDTSTART:20261020T100000Z\r\nDTEND:20261020T110000Z\r\nEND:VEVENT\r\n"
	}
	snapshot := func(events string) io.Reader {
		return strings.NewReader("BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" + events + "END:VCALENDAR\r\n")
	}
	_, err = service.importFeedSnapshot(other, "", snapshot(event("UID:fixed\r\n", "Other")))
	require.NoError(t, err)
	var fixedID int64
	for i := 0; i < 3; i++ {
		count, err := service.importFeedSnapshot(feed, "", snapshot(
			event("", "No UID")+event(fmt.Sprintf("UID:rotating-%d\r\n", i), "Rotating")+event("UID:fixed\r\n", "Fixed")))
		require.NoError(t, err)
		assert.Equal(t, 3, count)
		all, err := repo.ListAll(nil)
		require.NoError(t, err)
		assert.Len(t, all, 5)
		for _, ev := range all {
			if ev.Title == "Fixed" || ev.Title == "Local edit" {
				if fixedID == 0 {
					fixedID = ev.ID
					ev.Title = "Local edit"
					ev.NoteSlug = "my-note"
					require.NoError(t, repo.Update(&ev))
				} else {
					assert.Equal(t, fixedID, ev.ID)
					assert.Equal(t, "Local edit", ev.Title)
					assert.Equal(t, "my-note", ev.NoteSlug)
				}
			}
		}
	}
	for _, bad := range []string{"", "upstream error", "BEGIN:VCALENDAR\r\n", "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"} {
		_, err = service.importFeedSnapshot(feed, "", strings.NewReader(bad))
		require.Error(t, err)
	}
	// Invalid, oversized and failed writes must preserve the successful snapshot.
	_, err = service.importFeedSnapshot(feed, "", snapshot(strings.ReplaceAll(event("", "Invalid"), "DTEND:20261020T110000Z", "DTEND:20261020T090000Z")))
	require.Error(t, err)
	_, err = service.importFeedSnapshot(feed, "", strings.NewReader(strings.Repeat("x", maxFeedImportSize+1)))
	require.ErrorIs(t, err, errFeedLimit)
	_, err = service.importFeedSnapshot(feed, "", snapshot(strings.Repeat(event("", "Many"), maxFeedEvents+1)))
	require.ErrorIs(t, err, errFeedLimit)
	_, err = db.Exec("CREATE TRIGGER reject_feed_insert BEFORE INSERT ON events BEGIN SELECT RAISE(ABORT, 'test failure'); END")
	require.NoError(t, err)
	_, err = service.importFeedSnapshot(feed, "", snapshot(event("", "Replacement")))
	require.Error(t, err)
	all, err := repo.ListAll(nil)
	require.NoError(t, err)
	assert.Len(t, all, 5)
	_, err = db.Exec("DROP TRIGGER reject_feed_insert")
	require.NoError(t, err)
	count, err := service.importFeedSnapshot(feed, "", snapshot(""))
	require.NoError(t, err)
	assert.Zero(t, count)
	all, err = repo.ListAll(nil)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestFeedRefreshDoesNotOverwriteChangedSubscription(t *testing.T) {
	db, err := repository.OpenDB(repository.MemoryDSN(t.Name()), 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo, err := repository.NewSQLiteRepository(db)
	require.NoError(t, err)
	feed := &model.Feed{URL: "https://example.com/old", RefreshIntervalMinutes: 60}
	require.NoError(t, repo.CreateFeed(feed))
	stale := *feed
	feed.URL = "https://example.com/new"
	feed.Enabled = false
	require.NoError(t, repo.UpdateFeed(feed))
	require.Error(t, repo.ReplaceFeedEvents(&stale, nil))
	stale.LastError = "stale response"
	require.NoError(t, repo.UpdateFeedRefreshStatus(&stale))
	actual, err := repo.GetFeedByID(feed.ID)
	require.NoError(t, err)
	assert.Equal(t, feed.URL, actual.URL)
	assert.False(t, actual.Enabled)
	assert.Empty(t, actual.LastError)
}

func TestFeedUIDCollisionsAreScopedToFeed(t *testing.T) {
	db, err := repository.OpenDB(repository.MemoryDSN(t.Name()), 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo, err := repository.NewSQLiteRepository(db)
	require.NoError(t, err)
	service := NewFeedService(repo, repo, repo)
	work := &model.Calendar{Name: "Work"}
	require.NoError(t, repo.CreateCalendar(work))
	const uid = "work-meeting-123@corp"
	manual := &model.Event{
		Title: "Manual import", IcsUID: uid, CalendarID: work.ID,
		StartTime: "2026-10-20T10:00:00Z", EndTime: "2026-10-20T11:00:00Z",
	}
	require.NoError(t, repo.Create(manual))
	feeds := []*model.Feed{
		{URL: "https://example.com/misc"},
		{URL: "https://example.com/work", CalendarID: work.ID},
		{URL: "https://example.com/work-other", CalendarID: work.ID},
	}
	for _, feed := range feeds {
		require.NoError(t, repo.CreateFeed(feed))
	}
	refresh := func(feed *model.Feed, title string) {
		t.Helper()
		body := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:" + uid +
			"\r\nSUMMARY:" + title + "\r\nDTSTART:20261020T100000Z\r\nDTEND:20261020T110000Z\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
		count, err := service.importFeedSnapshot(feed, "", strings.NewReader(body))
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	}
	for i, feed := range feeds {
		refresh(feed, fmt.Sprintf("Feed %d", i))
	}
	all, err := repo.ListAll(nil)
	require.NoError(t, err)
	require.Len(t, all, 4)
	byTitle := make(map[string]model.Event)
	for _, event := range all {
		assert.Equal(t, uid, event.IcsUID)
		byTitle[event.Title] = event
	}
	assert.Equal(t, int64(0), byTitle["Feed 0"].CalendarID)
	assert.Equal(t, work.ID, byTitle["Feed 1"].CalendarID)
	assert.Equal(t, work.ID, byTitle["Feed 2"].CalendarID)
	assert.Equal(t, manual.ID, byTitle["Manual import"].ID)
	// Subsequent refreshes retain each feed's own row, despite colliding UIDs.
	for _, feed := range feeds {
		refresh(feed, "Upstream change")
	}
	after, err := repo.ListAll(nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, all, after)
	// Removing one feed's event must leave all other sources intact.
	_, err = service.importFeedSnapshot(feeds[0], "", strings.NewReader("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n"))
	require.NoError(t, err)
	remaining, err := repo.ListAll([]int64{work.ID})
	require.NoError(t, err)
	assert.Len(t, remaining, 3)
	all, err = repo.ListAll(nil)
	require.NoError(t, err)
	assert.ElementsMatch(t, remaining, all)
}
