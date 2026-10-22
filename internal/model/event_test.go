package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	for _, tc := range []struct {
		input   string
		seconds int64
	}{
		{"PT1H", 3600}, {"PT30M", 1800}, {"P1D", 86400}, {"P2W", 1209600},
		{"P1DT2H30M5S", 95405}, {"pt1h", 3600}, {"PT0H1S", 1},
		{"PT9223372036S", 9223372036}, {"P106751DT23H47M16S", 9223372036},
		{"PT00001S", 1},
	} {
		t.Run(tc.input, func(t *testing.T) {
			actual, err := ParseDuration(tc.input)
			require.NoError(t, err)
			assert.Equal(t, time.Duration(tc.seconds)*time.Second, actual)
		})
	}
}

func TestParseDurationRejectsMalformedAndOverflow(t *testing.T) {
	for _, input := range []string{
		"", "1H", "P", "PT", "PD", "P1", "PT1H5", "PTTT1H", "P1DT", "P1T2H",
		"P1DTT1H", "PT1TH", "PT1M1H", "PT1H1H", "P1D1D", "P1W1D", "P1WT1H",
		"PT1.5H", "PT-1H", "PT+1H", "P1Y", "P1M", "PT1D", "P0D", "PT0S", "PT1H\n",
		"PT9223372037S", "P106752D", "PT2562048H", "PT153722868M", "P15251W",
		"P106751DT23H47M17S", "PT9223372036854775807H", "PT18446744073709551616S",
	} {
		t.Run(input, func(t *testing.T) {
			actual, err := ParseDuration(input)
			require.Error(t, err)
			assert.Zero(t, actual)
		})
	}
}
