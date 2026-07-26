package model

import (
	"fmt"
	"regexp"
)

// MaxNoteSlugLength matches the slug length limit of the MyNotes API.
const MaxNoteSlugLength = 100

// noteSlugRe is the MyNotes slug grammar (see its openapi.yaml): lowercase
// alphanumeric words joined by single hyphens.
var noteSlugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidateNoteSlug checks the slug of a linked MyNotes note. The empty string
// means "no note linked" and is always valid. The note itself is never fetched:
// MyCal stores the slug, the browser resolves it against MyNotes.
func ValidateNoteSlug(s string) error {
	if s == "" {
		return nil
	}
	if len(s) > MaxNoteSlugLength {
		return fmt.Errorf("note_slug must be at most %d characters", MaxNoteSlugLength)
	}
	if !noteSlugRe.MatchString(s) {
		return fmt.Errorf("note_slug must be a lowercase alphanumeric slug, e.g. my-note")
	}
	return nil
}
