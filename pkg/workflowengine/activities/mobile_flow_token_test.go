// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/signedtoken"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestSignMobileFlowToken(t *testing.T) {
	app := newCredimiTestApp(t)
	act := NewSignMobileFlowTokenActivity(app)

	t.Run("refuses runner namespaces", func(t *testing.T) {
		_, err := act.sign("runner-abcdefghijklmno", "wf", "run")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no mobile flow token for runner namespace")
		var appErr *temporal.ApplicationError
		require.True(t, errors.As(err, &appErr))
		assert.True(t, appErr.NonRetryable())
	})

	t.Run("signs the calling run", func(t *testing.T) {
		token, err := act.sign("usera-s-organization", "wf", "run")
		require.NoError(t, err)

		var grant MobileFlowGrant
		require.NoError(t, signedtoken.Verify(app, MobileFlowTokenLabel, token, &grant))
		assert.Equal(t, MobileFlowGrant{
			Namespace:  "usera-s-organization",
			WorkflowID: "wf",
			RunID:      "run",
		}, grant)

		var other MobileFlowGrant
		require.Error(t, signedtoken.Verify(
			app,
			"credimi/action-author/approval/v1",
			token,
			&other,
		))
	})
}
