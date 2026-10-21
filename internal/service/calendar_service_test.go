package service

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

type deleteCalendarRepo struct {
	mockCalRepo
	err    error
	called bool
}

func (r *deleteCalendarRepo) DeleteCalendar(id int64) error { r.called = true; return r.err }

func TestCalendarDelete(t *testing.T) {
	r := &deleteCalendarRepo{}
	s := NewCalendarService(r)
	assert.ErrorIs(t, s.Delete(0), ErrValidation)
	assert.False(t, r.called)
	assert.NoError(t, s.Delete(1))
	assert.True(t, r.called)
	r.err = sql.ErrNoRows
	assert.ErrorIs(t, s.Delete(99), ErrNotFound)
}
