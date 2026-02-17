package model

import (
	"fmt"
	"time"
)

type Event struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Color       string `json:"color"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Color       string `json:"color"`
}

func (r *CreateEventRequest) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if r.StartTime == "" {
		return fmt.Errorf("start_time is required")
	}
	if r.EndTime == "" {
		return fmt.Errorf("end_time is required")
	}
	start, err := time.Parse(time.RFC3339, r.StartTime)
	if err != nil {
		return fmt.Errorf("start_time must be RFC 3339 format")
	}
	end, err := time.Parse(time.RFC3339, r.EndTime)
	if err != nil {
		return fmt.Errorf("end_time must be RFC 3339 format")
	}
	if !end.After(start) {
		return fmt.Errorf("end_time must be after start_time")
	}
	return nil
}

type UpdateEventRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	StartTime   *string `json:"start_time"`
	EndTime     *string `json:"end_time"`
	Color       *string `json:"color"`
}

func (r *UpdateEventRequest) Validate() error {
	if r.Title != nil && *r.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	start := ""
	end := ""
	if r.StartTime != nil {
		if _, err := time.Parse(time.RFC3339, *r.StartTime); err != nil {
			return fmt.Errorf("start_time must be RFC 3339 format")
		}
		start = *r.StartTime
	}
	if r.EndTime != nil {
		if _, err := time.Parse(time.RFC3339, *r.EndTime); err != nil {
			return fmt.Errorf("end_time must be RFC 3339 format")
		}
		end = *r.EndTime
	}
	if start != "" && end != "" {
		s, _ := time.Parse(time.RFC3339, start)
		e, _ := time.Parse(time.RFC3339, end)
		if !e.After(s) {
			return fmt.Errorf("end_time must be after start_time")
		}
	}
	return nil
}
