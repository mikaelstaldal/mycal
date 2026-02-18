package repository

import (
	"database/sql"
	"testing"

	"github.com/mikaelstaldal/mycal/internal/model"
	_ "modernc.org/sqlite"
)

func setupTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	repo, err := NewSQLiteRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func createTestEvent(t *testing.T, repo *SQLiteRepository, title, desc, start, end string) *model.Event {
	t.Helper()
	e := &model.Event{
		Title:       title,
		Description: desc,
		StartTime:   start,
		EndTime:     end,
	}
	if err := repo.Create(e); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestSearchByTitle(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, "Team Meeting", "Weekly sync", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")
	createTestEvent(t, repo, "Lunch Break", "Cafeteria", "2026-02-18T12:00:00Z", "2026-02-18T13:00:00Z")

	results, err := repo.Search("meeting", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Team Meeting" {
		t.Errorf("expected 'Team Meeting', got %q", results[0].Title)
	}
}

func TestSearchByDescription(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, "Event A", "Important discussion about budgets", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")
	createTestEvent(t, repo, "Event B", "Casual chat", "2026-02-18T12:00:00Z", "2026-02-18T13:00:00Z")

	results, err := repo.Search("budgets", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Event A" {
		t.Errorf("expected 'Event A', got %q", results[0].Title)
	}
}

func TestSearchWithTimeRange(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, "Morning Meeting", "Standup", "2026-02-18T09:00:00Z", "2026-02-18T10:00:00Z")
	createTestEvent(t, repo, "Afternoon Meeting", "Review", "2026-02-18T15:00:00Z", "2026-02-18T16:00:00Z")

	results, err := repo.Search("meeting", "2026-02-18T14:00:00Z", "2026-02-18T17:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Afternoon Meeting" {
		t.Errorf("expected 'Afternoon Meeting', got %q", results[0].Title)
	}
}

func TestSearchAfterUpdate(t *testing.T) {
	repo := setupTestRepo(t)
	e := createTestEvent(t, repo, "Old Title", "Description", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")

	e.Title = "New Title"
	if err := repo.Update(e); err != nil {
		t.Fatal(err)
	}

	results, err := repo.Search("Old", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for old title, got %d", len(results))
	}

	results, err = repo.Search("New", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for new title, got %d", len(results))
	}
}

func TestSearchAfterDelete(t *testing.T) {
	repo := setupTestRepo(t)
	e := createTestEvent(t, repo, "Deletable Event", "Will be removed", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")

	if err := repo.Delete(e.ID); err != nil {
		t.Fatal(err)
	}

	results, err := repo.Search("Deletable", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results after delete, got %d", len(results))
	}
}

func TestSearchSpecialCharacters(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, `Event with "quotes"`, "Has special chars: AND OR NOT", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")

	results, err := repo.Search(`"quotes"`, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// FTS5 operators should be safely quoted
	results, err = repo.Search("AND OR NOT", "", "")
	if err != nil {
		t.Fatal(err)
	}
	// Should not error even with FTS5 operator-like terms
}

func TestSearchEmptyQuery(t *testing.T) {
	repo := setupTestRepo(t)
	createTestEvent(t, repo, "Some Event", "Description", "2026-02-18T10:00:00Z", "2026-02-18T11:00:00Z")

	results, err := repo.Search("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Fatalf("expected nil results for empty query, got %d", len(results))
	}
}
