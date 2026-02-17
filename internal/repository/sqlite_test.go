package repository

import (
	"database/sql"
	"testing"

	"github.com/mikaelstaldal/mycal/internal/model"
)

func newTestRepo(t *testing.T) *SQLiteRepository {
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

func TestCreateAndGetByID(t *testing.T) {
	repo := newTestRepo(t)
	e := &model.Event{
		Title:     "Test Event",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T11:00:00Z",
	}
	if err := repo.Create(e); err != nil {
		t.Fatal(err)
	}
	if e.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if e.CreatedAt == "" {
		t.Fatal("expected created_at to be set")
	}

	got, err := repo.GetByID(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Test Event" {
		t.Fatalf("got title %q, want %q", got.Title, "Test Event")
	}
}

func TestGetByIDNotFound(t *testing.T) {
	repo := newTestRepo(t)
	got, err := repo.GetByID(999)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("expected nil for missing event")
	}
}

func TestList(t *testing.T) {
	repo := newTestRepo(t)
	events := []model.Event{
		{Title: "Jan Event", StartTime: "2026-01-10T10:00:00Z", EndTime: "2026-01-10T11:00:00Z"},
		{Title: "Feb Event", StartTime: "2026-02-15T10:00:00Z", EndTime: "2026-02-15T11:00:00Z"},
		{Title: "Mar Event", StartTime: "2026-03-20T10:00:00Z", EndTime: "2026-03-20T11:00:00Z"},
	}
	for i := range events {
		if err := repo.Create(&events[i]); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repo.List("2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	if got[0].Title != "Feb Event" {
		t.Fatalf("got title %q, want %q", got[0].Title, "Feb Event")
	}
}

func TestUpdate(t *testing.T) {
	repo := newTestRepo(t)
	e := &model.Event{
		Title:     "Original",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T11:00:00Z",
	}
	if err := repo.Create(e); err != nil {
		t.Fatal(err)
	}
	e.Title = "Updated"
	if err := repo.Update(e); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetByID(e.ID)
	if got.Title != "Updated" {
		t.Fatalf("got title %q, want %q", got.Title, "Updated")
	}
}

func TestDelete(t *testing.T) {
	repo := newTestRepo(t)
	e := &model.Event{
		Title:     "To Delete",
		StartTime: "2026-03-15T10:00:00Z",
		EndTime:   "2026-03-15T11:00:00Z",
	}
	if err := repo.Create(e); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(e.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetByID(e.ID)
	if got != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestDeleteNotFound(t *testing.T) {
	repo := newTestRepo(t)
	err := repo.Delete(999)
	if err == nil {
		t.Fatal("expected error for missing event")
	}
}
