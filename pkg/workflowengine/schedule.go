// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"time"

	"go.temporal.io/sdk/client"
)

// ScheduleMode represents the scheduling configuration
type ScheduleMode struct {
	Mode string `json:"mode"` // "daily", "weekly", "monthly"
	Day  *int   `json:"day"`  // 0-6 for weekly (0=Sunday), 0-30 for monthly
}

func BuildCalendarSpec(mode ScheduleMode) []client.ScheduleCalendarSpec {
	now := time.Now()
	switch mode.Mode {
	case "daily":
		// Run every day at the specified hour
		return []client.ScheduleCalendarSpec{{
			Second: []client.ScheduleRange{{Start: now.Second()}},
			Minute: []client.ScheduleRange{{Start: now.Minute()}},
			Hour:   []client.ScheduleRange{{Start: now.Hour()}},
			Month:  []client.ScheduleRange{{Start: 1, End: 12}},
			DayOfMonth: []client.ScheduleRange{{
				Start: 1, End: 31,
			}},
		}}

	case "weekly":
		// Run every day at the specified hour
		return []client.ScheduleCalendarSpec{{
			Second:    []client.ScheduleRange{{Start: now.Second()}},
			Minute:    []client.ScheduleRange{{Start: now.Minute()}},
			Hour:      []client.ScheduleRange{{Start: now.Hour()}},
			DayOfWeek: []client.ScheduleRange{{Start: *mode.Day}},
			Month:     []client.ScheduleRange{{Start: 1, End: 12}},
		}}

	case "monthly":
		// Run every day at the specified hour
		return buildMonthlyCalendarSpecs(mode)

	default:
		return nil
	}
}
func buildMonthlyCalendarSpecs(mode ScheduleMode) []client.ScheduleCalendarSpec {
	now := time.Now()
	dayIndex := *mode.Day + 1

	if dayIndex <= 28 {
		return []client.ScheduleCalendarSpec{{
			Second: []client.ScheduleRange{{Start: now.Second()}},
			Minute: []client.ScheduleRange{{Start: now.Minute()}},
			Hour:   []client.ScheduleRange{{Start: now.Hour()}},
			Month:  []client.ScheduleRange{{Start: 1, End: 12}},
			DayOfMonth: []client.ScheduleRange{{
				Start: dayIndex,
			}},
		}}
	}

	maxDays := func(m time.Month) int {
		switch m {
		case time.January,
			time.March,
			time.May,
			time.July,
			time.August,
			time.October,
			time.December:
			return 31
		case time.April, time.June, time.September, time.November:
			return 30
		default:
			return 28
		}
	}

	specs := make([]client.ScheduleCalendarSpec, 0, 12)

	for m := time.January; m <= time.December; m++ {
		maxDay := maxDays(m)
		d := dayIndex
		if d > maxDay {
			d = maxDay
		}

		specs = append(specs, client.ScheduleCalendarSpec{
			Second: []client.ScheduleRange{{Start: now.Second()}},
			Minute: []client.ScheduleRange{{Start: now.Minute()}},
			Hour:   []client.ScheduleRange{{Start: now.Hour()}},
			Month:  []client.ScheduleRange{{Start: int(m)}},
			DayOfMonth: []client.ScheduleRange{{
				Start: d,
			}},
		})
	}

	return specs
}

func ParseScheduleMode(calendars []client.ScheduleCalendarSpec) ScheduleMode {
	mode := ScheduleMode{}

	if len(calendars) == 0 {
		return mode
	}

	if isDaily(calendars) {
		mode.Mode = "daily"
		mode.Day = nil
		return mode
	}

	if isWeekly(calendars) {
		mode.Mode = "weekly"
		v := calendars[0].DayOfWeek[0].Start
		mode.Day = &v
		return mode
	}

	mode.Mode = "monthly"
	v := extractMonthlyDay(calendars)
	mode.Day = &v
	return mode
}

func isDaily(cals []client.ScheduleCalendarSpec) bool {
	if len(cals) != 1 {
		return false
	}
	c := cals[0]

	// Month = 1..12
	if len(c.Month) != 1 || c.Month[0].Start != 1 || c.Month[0].End != 12 {
		return false
	}

	// DayOfMonth = 1..31
	if len(c.DayOfMonth) != 1 || c.DayOfMonth[0].Start != 1 || c.DayOfMonth[0].End != 31 {
		return false
	}

	// DayOfWeek = 0..6
	if len(c.DayOfWeek) != 1 || c.DayOfWeek[0].Start != 0 || c.DayOfWeek[0].End != 6 {
		return false
	}

	return true
}

func isWeekly(cals []client.ScheduleCalendarSpec) bool {
	if len(cals) != 1 {
		return false
	}
	c := cals[0]

	// DayOfMonth = 1..31
	if len(c.DayOfMonth) != 1 || c.DayOfMonth[0].Start != 1 || c.DayOfMonth[0].End != 31 {
		return false
	}

	if len(c.DayOfWeek) != 1 || c.DayOfWeek[0].Start != 0 || c.DayOfWeek[0].End != 6 {
		return true
	}

	return false
}

func extractMonthlyDay(cals []client.ScheduleCalendarSpec) int {
	maxDay := 1
	for _, c := range cals {
		if len(c.DayOfMonth) == 0 {
			continue
		}
		d := c.DayOfMonth[0].Start
		if d > maxDay {
			maxDay = d
		}
	}
	return maxDay - 1
}
