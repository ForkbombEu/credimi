// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"context"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"go.temporal.io/sdk/client"
)

// ScheduleMode represents the scheduling configuration
type ScheduleMode struct {
	Mode string `json:"mode"` // "daily", "weekly", "monthly"
	Day  *int   `json:"day"`  // 0-6 for weekly (0=Sunday), 0-30 for monthly
}

type ScheduleStartInfo struct {
	ID string
}

func StartScheduledWorkflowWithOptions(
	runInfo WorkflowRunInfo,
	workflowID, namespace string,
	scheduleMode ScheduleMode,
	timeZone string,
) (ScheduleStartInfo, error) {
	c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		return ScheduleStartInfo{}, fmt.Errorf(
			"unable to create Temporal client for namespace %q: %w",
			namespace,
			err,
		)
	}

	ctx := context.Background()
	scheduleID := fmt.Sprintf("Schedule_ID_%s", workflowID)

	calendarSpec := buildCalendarSpec(scheduleMode)
	scheduleHandle, err := c.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			Calendars:    calendarSpec,
			TimeZoneName: timeZone,
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("Scheduled_%s", workflowID),
			Workflow:  runInfo.Name,
			TaskQueue: runInfo.TaskQueue,
			Args:      []any{runInfo.Input},
			Memo:      runInfo.Memo,
		},
		Memo: map[string]any{
			"test":                 runInfo.Memo["test"],
			"original_workflow_id": workflowID,
		},
	})
	if err != nil {
		return ScheduleStartInfo{}, fmt.Errorf(
			"failed to start scheduledID from workflowID: %s: %w",
			workflowID,
			err,
		)
	}

	_, err = scheduleHandle.Describe(ctx)
	if err != nil {
		return ScheduleStartInfo{}, fmt.Errorf(
			"failed to describe scheduledID: %s: %w",
			scheduleID,
			err,
		)
	}

	return ScheduleStartInfo{
		ID: scheduleID,
	}, nil
}
func buildCalendarSpec(mode ScheduleMode) []client.ScheduleCalendarSpec {
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
