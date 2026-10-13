// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package realtimelogs

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestNotifyLogsUpdateNoSubscribers(t *testing.T) {
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	defer app.Cleanup()

	err = Notify(app, "subscription-1", []map[string]any{{"step": "ok"}})
	require.NoError(t, err)
}
