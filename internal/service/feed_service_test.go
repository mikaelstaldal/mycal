package service

import (
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
