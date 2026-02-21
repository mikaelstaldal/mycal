package service

import (
	"sort"
	"time"

	"github.com/mikaelstaldal/mycal/internal/model"
)

const maxExpansions = 1000

func expandRecurring(event model.Event, from, to time.Time) []model.Event {
	if event.RecurrenceFreq == "" {
		return nil
	}

	startTime, err := time.Parse(time.RFC3339, event.StartTime)
	if err != nil {
		return nil
	}
	endTime, err := time.Parse(time.RFC3339, event.EndTime)
	if err != nil {
		return nil
	}
	duration := endTime.Sub(startTime)

	var instances []model.Event
	for i := 0; i < maxExpansions; i++ {
		if event.RecurrenceCount > 0 && i >= event.RecurrenceCount {
			break
		}

		instanceStart := addFreq(startTime, event.RecurrenceFreq, i)
		instanceEnd := instanceStart.Add(duration)

		// Stop if past the query window
		if instanceStart.Compare(to) >= 0 {
			break
		}

		// Include if overlaps the query window
		if instanceEnd.After(from) {
			inst := event
			inst.StartTime = instanceStart.Format(time.RFC3339)
			inst.EndTime = instanceEnd.Format(time.RFC3339)
			inst.RecurrenceIndex = i
			instances = append(instances, inst)
		}
	}
	return instances
}

func addFreq(t time.Time, freq string, n int) time.Time {
	switch freq {
	case "DAILY":
		return t.AddDate(0, 0, n)
	case "WEEKLY":
		return t.AddDate(0, 0, n*7)
	case "MONTHLY":
		return t.AddDate(0, n, 0)
	case "YEARLY":
		return t.AddDate(n, 0, 0)
	default:
		return t
	}
}

func mergeEvents(a, b []model.Event) []model.Event {
	result := append(a, b...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime < result[j].StartTime
	})
	return result
}
