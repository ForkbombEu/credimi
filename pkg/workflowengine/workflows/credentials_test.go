// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

const rawJSON = `{
    "credential_issuer": "testissuer",
    "credential_endpoint": "testendpoint",
    "credential_configurations_supported": {
        "test": {
		}
    }
}

`

func Test_Workflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	CheckActivity := activities.NewCheckCredentialsIssuerActivity()
	env.RegisterActivityWithOptions(CheckActivity.Execute, activity.RegisterOptions{
		Name: CheckActivity.Name(),
	})
	JSONActivity := activities.NewJSONActivity(map[string]reflect.Type{
		"DummyStruct": reflect.TypeOf(nil),
	})
	env.RegisterActivityWithOptions(JSONActivity.Execute, activity.RegisterOptions{
		Name: JSONActivity.Name(),
	})
	ValidateActivity := activities.NewSchemaValidationActivity()
	env.RegisterActivityWithOptions(ValidateActivity.Execute, activity.RegisterOptions{
		Name: ValidateActivity.Name(),
	})
	HTTPActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
		Name: HTTPActivity.Name(),
	})
	var credentialWorkflow CredentialsIssuersWorkflow
	var JSONOutput map[string]any

	err := json.Unmarshal([]byte(rawJSON), &JSONOutput)
	require.NoError(t, err)
	// Mock activity implementation
	env.OnActivity(CheckActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"rawJSON":  rawJSON,
			"source":   "test",
			"base_url": "testURL"},
		},
			nil)
	env.OnActivity(JSONActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: JSONOutput}, nil)
	env.OnActivity(ValidateActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, nil)
	env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"body": map[string]any{"key": "test-credential"}}}, nil)
	env.ExecuteWorkflow(credentialWorkflow.Workflow, workflowengine.WorkflowInput{
		Config: map[string]any{
			"app_url": "test.app",
			"issuer_schema": `{
					"type": "object",
					"properties": {
						"name": { "type": "string" }
					},
					"required": ["name"]
				}`,
			"namespace": "test_namespace",
		},
		Payload: map[string]any{
			"issuerID": "test_issuer",
			"base_url": "test_url",
		},
	})

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(
		t,
		"Successfully retrieved and stored and update credentials from 'test'",
		result.Message,
	)
	require.Equal(
		t,
		map[string]any{
			"RemovedCredentials": []any{interface{}(nil)},
			"StoredCredentials":  []any{interface{}("test-credential")},
		},
		result.Log,
	)
}
