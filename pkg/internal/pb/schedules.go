// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/pocketbase/pocketbase/core"
)

type ScheduleStatus struct {
	DisplayName    string `json:"display_name,omitempty"`
	NextActionTime string `json:"next_action_time,omitempty"`
	Paused         bool   `json:"paused"`
}

func RegisterSchedulesHooks(app core.App) {
	app.OnRecordEnrich("schedules").BindFunc(func(e *core.RecordEnrichEvent) error {
		ownerID := e.Record.GetString("owner")

		owner, err := e.App.FindRecordById("organizations", ownerID)
		if err != nil {
			return fmt.Errorf("failed to fetch owner organization: %w", err)
		}

		namespace := owner.GetString("canonified_name")

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return fmt.Errorf(
				"unable to create Temporal client for namespace %q: %w",
				namespace,
				err,
			)
		}
		ctx := context.Background()
		handle := c.ScheduleClient().GetHandle(ctx, e.Record.GetString("temporal_schedule_id"))
		desc, err := handle.Describe(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe schedule: %w", err)
		}
		var displayName string
		if desc.Memo != nil {
			if field, ok := desc.Memo.Fields["test"]; ok {
				displayName = handlers.DecodeFromTemporalPayload(string(field.Data))
			}
		}

		status := ScheduleStatus{
			DisplayName:    displayName,
			NextActionTime: desc.Info.NextActionTimes[0].Format("02/01/2006, 15:04:05"),
			Paused:         desc.Schedule.State.Paused,
		}
		e.Record.WithCustomData(true)
		e.Record.Set("__schedule_status__", status)

		return e.Next()

	})

}
