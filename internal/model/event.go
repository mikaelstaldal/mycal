package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Event struct {
	ID                      int64
	StringID                string
	ParentID                string
	Title                   string
	Description             string
	StartTime               string
	EndTime                 string
	AllDay                  bool
	Color                   string
	RecurrenceFreq          string
	RecurrenceCount         int
	RecurrenceUntil         string
	RecurrenceInterval      int
	RecurrenceByDay         string
	RecurrenceByMonthDay    string
	RecurrenceByMonth       string
	ExDates                 string
	RDates                  string
	RecurrenceIndex         int
	RecurrenceParentID      *int64
	RecurrenceOriginalStart string
	Duration                string
	Categories              string
	URL                     string
	ReminderMinutes         int
	Location                string
	Latitude                *float64
	Longitude               *float64
	CalendarID              int64
	CalendarName            string
	IcsUID                  string
	CreatedAt               string
	UpdatedAt               string
	ImportUID               string // transient field for iCal import UID matching
}

func (e *Event) IsRecurring() bool {
	return e.RecurrenceFreq != ""
}

const MaxCalendarNameLength = 100

// ParseDuration parses an ISO 8601 duration string like PT1H, PT30M, P1D, P1DT2H30M.
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	s = strings.ToUpper(s)
	if !strings.HasPrefix(s, "P") {
		return 0, fmt.Errorf("duration must start with P")
	}
	s = s[1:] // strip "P"

	var total time.Duration
	inTime := false

	num := ""
	for _, c := range s {
		if c == 'T' {
			inTime = true
			continue
		}
		if c >= '0' && c <= '9' {
			num += string(c)
			continue
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, fmt.Errorf("invalid duration number: %s", num)
		}
		num = ""

		if inTime {
			switch c {
			case 'H':
				total += time.Duration(n) * time.Hour
			case 'M':
				total += time.Duration(n) * time.Minute
			case 'S':
				total += time.Duration(n) * time.Second
			default:
				return 0, fmt.Errorf("unknown time unit: %c", c)
			}
		} else {
			switch c {
			case 'D':
				total += time.Duration(n) * 24 * time.Hour
			case 'W':
				total += time.Duration(n) * 7 * 24 * time.Hour
			default:
				return 0, fmt.Errorf("unknown date unit: %c", c)
			}
		}
	}

	if total <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}
	return total, nil
}

// FormatEventID builds a composite string ID.
// For recurring instances: "parentID_instanceStart".
// For simple events: "dbID".
func FormatEventID(dbID int64, instanceStart string) string {
	if instanceStart != "" {
		return strconv.FormatInt(dbID, 10) + "_" + instanceStart
	}
	return strconv.FormatInt(dbID, 10)
}

// ParseEventID parses a composite string ID into a DB ID and optional instance start.
func ParseEventID(s string) (dbID int64, instanceStart string, err error) {
	idx := strings.Index(s, "_")
	if idx < 0 {
		dbID, err = strconv.ParseInt(s, 10, 64)
		return
	}
	dbID, err = strconv.ParseInt(s[:idx], 10, 64)
	if err != nil {
		return
	}
	instanceStart = s[idx+1:]
	return
}

// SetStringID computes the StringID and ParentID from internal state.
func (e *Event) SetStringID() {
	if e.RecurrenceParentID != nil {
		// Override instance: parentID_originalStart
		parentStr := strconv.FormatInt(*e.RecurrenceParentID, 10)
		e.StringID = parentStr + "_" + e.RecurrenceOriginalStart
		e.ParentID = parentStr
	} else if e.RecurrenceFreq != "" && e.RecurrenceIndex > 0 {
		// Expanded recurring instance: ID_startTime
		parentStr := strconv.FormatInt(e.ID, 10)
		e.StringID = parentStr + "_" + e.StartTime
		e.ParentID = parentStr
	} else {
		// Non-recurring or first instance of recurring parent
		e.StringID = strconv.FormatInt(e.ID, 10)
	}
}
