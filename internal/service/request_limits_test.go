package service

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mikaelstaldal/mycal/internal/api"
	"github.com/stretchr/testify/require"
)

// Exercise the generated validators: service checks alone cannot pin the published contract.
func TestEventRequestSchemaLimits(t *testing.T) {
	for field, limit := range map[string]int{
		"Color": 100, "RecurrenceUntil": 100, "Duration": 100,
		"RecurrenceByDay": 5000, "RecurrenceByMonthday": 5000, "RecurrenceByMonth": 5000,
		"Exdates": 5000, "Rdates": 5000,
	} {
		t.Run(field, func(t *testing.T) {
			for _, size := range []int{limit, limit + 1} {
				create := validCreateReq()
				update := &api.UpdateEventRequest{}
				for _, req := range []interface{ Validate() error }{create, update} {
					reflect.ValueOf(req).Elem().FieldByName(field).Set(reflect.ValueOf(api.NewOptString(strings.Repeat("x", size))))
					if size == limit {
						require.NoError(t, req.Validate())
					} else {
						require.Error(t, req.Validate())
					}
				}
			}
		})
	}
	for _, interval := range []int{999, 1000} {
		create := validCreateReq()
		create.RecurrenceInterval = api.NewOptInt(interval)
		update := &api.UpdateEventRequest{RecurrenceInterval: api.NewOptInt(interval)}
		for _, req := range []interface{ Validate() error }{create, update} {
			if interval == 999 {
				require.NoError(t, req.Validate())
			} else {
				require.Error(t, req.Validate())
			}
		}
	}
}
