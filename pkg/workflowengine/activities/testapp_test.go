// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

const testDataDir = "../../../test_pb_data"

const (
	testOrgAID        = "co35481b68u3zj3"
	testOrgANamespace = "usera-s-organization"
	testOrgBID        = "3u4982xn6ah0433"
)

// newCredimiTestApp returns a test app with canonified-name hooks registered.
func newCredimiTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	canonify.RegisterCanonifyHooks(app)
	return app
}

// executeActivity runs act in a Temporal test activity environment.
func executeActivity(
	t *testing.T,
	act workflowengine.ExecutableActivity,
	payload any,
) (workflowengine.ActivityResult, error) {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
	var result workflowengine.ActivityResult
	value, err := env.ExecuteActivity(act.Name(), workflowengine.ActivityInput{Payload: payload})
	if err != nil {
		return result, err
	}
	require.NoError(t, value.Get(&result))
	return result, nil
}

// requireActivityError asserts err is an application error with the code of
// errorcodes key code and the given retry behavior.
func requireActivityError(t *testing.T, err error, code string, nonRetryable bool) {
	t.Helper()
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, errorcodes.Codes[code].Code, appErr.Type())
	require.Equal(t, nonRetryable, appErr.NonRetryable())
}
