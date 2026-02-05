// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

func TestBuildCalendarSpecDaily(t *testing.T) {
	specs := BuildCalendarSpec(ScheduleMode{Mode: "daily"})
	require.Len(t, specs, 1)

	spec := specs[0]
	require.Len(t, spec.Month, 1)
	require.Equal(t, 1, spec.Month[0].Start)
	require.Equal(t, 12, spec.Month[0].End)

	require.Len(t, spec.DayOfMonth, 1)
	require.Equal(t, 1, spec.DayOfMonth[0].Start)
	require.Equal(t, 31, spec.DayOfMonth[0].End)
}

func TestBuildCalendarSpecWeekly(t *testing.T) {
	day := 2
	specs := BuildCalendarSpec(ScheduleMode{Mode: "weekly", Day: &day})
	require.Len(t, specs, 1)

	spec := specs[0]
	require.Len(t, spec.DayOfWeek, 1)
	require.Equal(t, day, spec.DayOfWeek[0].Start)
	require.True(t, spec.DayOfWeek[0].End == 0 || spec.DayOfWeek[0].End == day)

	require.Len(t, spec.Month, 1)
	require.Equal(t, 1, spec.Month[0].Start)
	require.Equal(t, 12, spec.Month[0].End)
}

func TestBuildCalendarSpecMonthlySingle(t *testing.T) {
	day := 27
	specs := BuildCalendarSpec(ScheduleMode{Mode: "monthly", Day: &day})
	require.Len(t, specs, 1)

	spec := specs[0]
	require.Len(t, spec.Month, 1)
	require.Equal(t, 1, spec.Month[0].Start)
	require.Equal(t, 12, spec.Month[0].End)

	require.Len(t, spec.DayOfMonth, 1)
	require.Equal(t, day+1, spec.DayOfMonth[0].Start)
}

func TestBuildCalendarSpecMonthlyEdgeDays(t *testing.T) {
	cases := []struct {
		name string
		day  int
	}{
		{name: "30th", day: 29},
		{name: "31st", day: 30},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			specs := BuildCalendarSpec(ScheduleMode{Mode: "monthly", Day: &tc.day})
			require.Len(t, specs, 12)
			dayIndex := tc.day + 1

			for _, spec := range specs {
				require.Len(t, spec.Month, 1)
				month := time.Month(spec.Month[0].Start)
				maxDay := maxDaysForTest(month)

				require.Len(t, spec.DayOfMonth, 1)
				day := spec.DayOfMonth[0].Start
				expected := dayIndex
				if expected > maxDay {
					expected = maxDay
				}
				require.Equal(t, expected, day)
			}
		})
	}
}

func TestParseScheduleMode(t *testing.T) {
	daily := []client.ScheduleCalendarSpec{calendarSpecForTest(1, 12, 1, 31, 0, 6)}
	parsed := ParseScheduleMode(daily)
	require.Equal(t, "daily", parsed.Mode)
	require.Nil(t, parsed.Day)

	weekly := []client.ScheduleCalendarSpec{calendarSpecForTest(1, 12, 1, 31, 2, 2)}
	parsed = ParseScheduleMode(weekly)
	require.Equal(t, "weekly", parsed.Mode)
	require.NotNil(t, parsed.Day)
	require.Equal(t, 2, *parsed.Day)

	monthly := []client.ScheduleCalendarSpec{
		{
			Month:      []client.ScheduleRange{{Start: 1}},
			DayOfMonth: []client.ScheduleRange{{Start: 30}},
		},
		{
			Month:      []client.ScheduleRange{{Start: 2}},
			DayOfMonth: []client.ScheduleRange{{Start: 28}},
		},
	}
	parsed = ParseScheduleMode(monthly)
	require.Equal(t, "monthly", parsed.Mode)
	require.NotNil(t, parsed.Day)
	require.Equal(t, 29, *parsed.Day)
}

func calendarSpecForTest(
	monthStart, monthEnd, domStart, domEnd, dowStart, dowEnd int,
) client.ScheduleCalendarSpec {
	spec := client.ScheduleCalendarSpec{}
	spec.Month = []client.ScheduleRange{{Start: monthStart, End: monthEnd}}
	spec.DayOfMonth = []client.ScheduleRange{{Start: domStart, End: domEnd}}
	spec.DayOfWeek = []client.ScheduleRange{{Start: dowStart, End: dowEnd}}
	return spec
}

func maxDaysForTest(m time.Month) int {
	switch m {
	case time.January, time.March, time.May, time.July, time.August, time.October, time.December:
		return 31
	case time.April, time.June, time.September, time.November:
		return 30
	default:
		return 28
	}
}
