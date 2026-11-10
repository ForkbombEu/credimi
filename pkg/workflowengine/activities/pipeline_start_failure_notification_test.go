// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
)

func TestNotifyPipelineStartFailureActivityUnknownOrganization(t *testing.T) {
	app := newPipelineResultsTestApp(t)

	_, err := executeActivity(
		t,
		NewNotifyPipelineStartFailureActivity(app),
		NotifyPipelineStartFailureInput{
			OwnerNamespace:     "no-such-organization",
			PipelineIdentifier: "no-such-organization/pipeline",
			TicketID:           "ticket-1",
		},
	)
	requireActivityError(t, err, errorcodes.RecordNotFound, true)
}
