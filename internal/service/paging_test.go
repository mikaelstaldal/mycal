package service_test

import (
	"testing"

	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/repository"
	"github.com/mikaelstaldal/mycal/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPagingAfterScopingAndExpansion(t *testing.T) {
	db, err := repository.OpenDB(repository.MemoryDSN(t.Name()), 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	repo, err := repository.NewSQLiteRepository(db)
	require.NoError(t, err)
	cal := model.Calendar{Name: "visible"}
	require.NoError(t, repo.CreateCalendar(&cal))
	hidden := model.Calendar{Name: "hidden"}
	require.NoError(t, repo.CreateCalendar(&hidden))
	for _, id := range []int64{hidden.ID, cal.ID, cal.ID} {
		e := model.Event{Title: "meeting", CalendarID: id, StartTime: "2026-03-10T10:00:00Z", EndTime: "2026-03-10T11:00:00Z"}
		require.NoError(t, repo.Create(&e))
	}
	recurring := model.Event{Title: "meeting recurring", CalendarID: cal.ID, StartTime: "2026-03-11T10:00:00Z", EndTime: "2026-03-11T11:00:00Z", RecurrenceFreq: "DAILY", RecurrenceCount: 3}
	require.NoError(t, repo.Create(&recurring))
	override := model.Event{Title: "meeting override", CalendarID: cal.ID,
		StartTime: "2026-03-10T10:00:00Z", EndTime: "2026-03-10T11:00:00Z",
		RecurrenceParentID: &recurring.ID, RecurrenceOriginalStart: "2026-03-12T10:00:00Z"}
	require.NoError(t, repo.Create(&override))
	for _, scoped := range []bool{false, true} {
		var reads repository.EventRepository = repo
		if scoped {
			reads = &repository.ScopedRepository{SQLiteRepository: repo, Allowed: map[int64]bool{cal.ID: true}}
		}
		svc := service.NewEventService(reads, repo)
		for _, search := range []bool{false, true} {
			var all []model.Event
			if search {
				all, err = svc.SearchPage("meeting", "", "", nil, 0, 0)
			} else {
				all, err = svc.ListPage("2026-03-01T00:00:00Z", "2026-04-01T00:00:00Z", nil, 0, 0)
			}
			require.NoError(t, err)
			expected := 6
			if search {
				expected = 5
			}
			if scoped {
				expected--
			}
			require.Len(t, all, expected)
			var combined []model.Event
			for offset := 0; offset < len(all)+2; offset += 2 {
				var page []model.Event
				if search {
					page, err = svc.SearchPage("meeting", "", "", nil, 2, offset)
				} else {
					page, err = svc.ListPage("2026-03-01T00:00:00Z", "2026-04-01T00:00:00Z", nil, 2, offset)
				}
				require.NoError(t, err)
				combined = append(combined, page...)
			}
			assert.Equal(t, all, combined)
			if scoped {
				for _, e := range combined {
					assert.Equal(t, cal.ID, e.CalendarID)
				}
			}
		}
	}
}
