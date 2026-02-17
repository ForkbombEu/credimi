// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestExtractCredimiJSON(t *testing.T) {
	t.Run("replaces credimi blocks and preserves functions", func(t *testing.T) {
		input := `
before
{{ credimi ` + "`" + `{"field_type":"string","field_default_value":"hello"}` + "`" + ` }}
middle
{{ credimi ` + "`" + `{"field_type":"string","field_default_value":"world"}` + "`" + ` uuidv4 }}
after
`
		out, err := extractCredimiJSON(input)
		require.NoError(t, err)
		require.NotContains(t, out, "{{")
		require.Contains(t, out, `"field_default_value":"hello"`)
		require.Contains(t, out, `"field_function":"uuidv4"`)
	})

	t.Run("invalid JSON reports error and keeps original", func(t *testing.T) {
		input := `{{ credimi ` + "`" + `{invalid` + "`" + ` }}`
		out, err := extractCredimiJSON(input)
		require.Error(t, err)
		require.Contains(t, out, input)
	})
}

func TestExtractValues(t *testing.T) {
	t.Run("extracts defaults and functions", func(t *testing.T) {
		node := map[string]any{
			"name": map[string]any{
				"field_type":          "string",
				"field_default_value": "Ada",
			},
			"payload": map[string]any{
				"field_type":          "object",
				"field_default_value": `{"x":1}`,
			},
			"uuid": map[string]any{
				"field_type":     "string",
				"field_function": "uuidv4",
			},
		}

		out := extractValues(node).(map[string]any)
		require.Equal(t, "Ada", out["name"])
		require.Equal(t, map[string]any{"x": float64(1)}, out["payload"])

		uuidStr, ok := out["uuid"].(string)
		require.True(t, ok)
		parsed, err := uuid.Parse(uuidStr)
		require.NoError(t, err)
		require.Equal(t, strings.ToLower(uuidStr), parsed.String())
	})

	t.Run("stringified field type is extracted", func(t *testing.T) {
		node := map[string]any{
			"value": `{"field_type":"string","field_default_value":"inline"}`,
		}
		out := extractValues(node).(map[string]any)
		require.Equal(t, "inline", out["value"])
	})

	t.Run("unknown function falls back to default value", func(t *testing.T) {
		node := map[string]any{
			"value": map[string]any{
				"field_type":          "string",
				"field_default_value": "fallback",
				"field_function":      "unknown",
			},
		}
		out := extractValues(node).(map[string]any)
		require.Equal(t, "fallback", out["value"])
	})
}

func TestConformanceCheckSetupHook(t *testing.T) {
	t.Run("openid suite populates payload and template", func(t *testing.T) {
		rootDir := t.TempDir()
		t.Setenv("ROOT_DIR", rootDir)

		checkID := "oid4vci/openid_conformance_suite/check1"
		configTemplate := `variant:
  credential_format: "jwt"
  client_id_prefix: "cid"
  request_method: "by_reference"
  response_mode: "direct_post"
form:
  foo: "bar"
test: "alpha"
`
		writeTemplateFile(t, rootDir, filepath.Join("config_templates", checkID+".yaml"), configTemplate)
		writeTemplateFile(
			t,
			rootDir,
			workflows.OpenIDNetStepCITemplatePathv1_0,
			"openid template",
		)

		result := runConformanceHookWorkflow(t, conformanceHookInput{
			CheckID: checkID,
			Config:  map[string]any{"user_mail": "test@example.com"},
		})

		payload := result.Payload
		require.Equal(t, "test@example.com", payload["user_mail"])
		require.Equal(t, "openid_conformance_suite", payload["suite"])
		require.Equal(t, checkID, payload["check_id"])
		require.Equal(t, "alpha", payload["test"])

		variantStr, ok := payload["variant"].(string)
		require.True(t, ok)
		var variant map[string]any
		require.NoError(t, json.Unmarshal([]byte(variantStr), &variant))
		require.Equal(t, "jwt", variant["credential_format"])
		require.Equal(t, "cid", variant["client_id_prefix"])
		require.Equal(t, "by_reference", variant["request_method"])
		require.Equal(t, "direct_post", variant["response_mode"])

		form, ok := payload["form"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "bar", form["foo"])

		memo, ok := result.Config["memo"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "openid_conformance_suite", memo["author"])
		require.Equal(t, "oid4vci", memo["standard"])
		require.Equal(t, "check1", memo["test"])
		require.Equal(t, "openid template", result.Config["template"])
	})

	t.Run("ewc suite populates session id", func(t *testing.T) {
		rootDir := t.TempDir()
		t.Setenv("ROOT_DIR", rootDir)

		checkID := "oid4vci/ewc/check2"
		configTemplate := `sessionId: "session-123"`
		writeTemplateFile(t, rootDir, filepath.Join("config_templates", checkID+".yaml"), configTemplate)
		writeTemplateFile(
			t,
			rootDir,
			filepath.Join(workflows.EWCTemplateFolderPath, "check2.yaml"),
			"ewc template",
		)

		result := runConformanceHookWorkflow(t, conformanceHookInput{
			CheckID: checkID,
			Config:  map[string]any{"user_mail": "test@example.com"},
		})

		payload := result.Payload
		require.Equal(t, "session-123", payload["session_id"])
		require.Equal(t, "ewc template", result.Config["template"])
	})

	t.Run("eudiw suite populates id and nonce", func(t *testing.T) {
		rootDir := t.TempDir()
		t.Setenv("ROOT_DIR", rootDir)

		checkID := "oid4vci/eudiw/check3"
		configTemplate := `id: "abc"
nonce: "xyz"
`
		writeTemplateFile(t, rootDir, filepath.Join("config_templates", checkID+".yaml"), configTemplate)
		writeTemplateFile(
			t,
			rootDir,
			filepath.Join(workflows.EudiwTemplateFolderPath, "check3.yaml"),
			"eudiw template",
		)

		result := runConformanceHookWorkflow(t, conformanceHookInput{
			CheckID: checkID,
			Config:  map[string]any{"user_mail": "test@example.com"},
		})

		payload := result.Payload
		require.Equal(t, "abc", payload["id"])
		require.Equal(t, "xyz", payload["nonce"])
		require.Equal(t, "eudiw template", result.Config["template"])
	})

	t.Run("missing user_mail returns error", func(t *testing.T) {
		rootDir := t.TempDir()
		t.Setenv("ROOT_DIR", rootDir)

		checkID := "oid4vci/openid_conformance_suite/check4"
		configTemplate := `variant: {}`
		writeTemplateFile(t, rootDir, filepath.Join("config_templates", checkID+".yaml"), configTemplate)
		writeTemplateFile(
			t,
			rootDir,
			workflows.OpenIDNetStepCITemplatePathv1_0,
			"openid template",
		)

		err := runConformanceHookWorkflowError(t, conformanceHookInput{
			CheckID: checkID,
			Config:  map[string]any{},
		})
		require.Error(t, err)
		var appErr *temporal.ApplicationError
		require.True(t, errors.As(err, &appErr))
		require.Equal(t, errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code, appErr.Type())
	})
}

func writeTemplateFile(t *testing.T, rootDir, relativePath, content string) {
	t.Helper()
	fullPath := filepath.Join(rootDir, relativePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
}

type conformanceHookInput struct {
	CheckID string
	Config  map[string]any
}

type conformanceHookResult struct {
	Payload map[string]any
	Config  map[string]any
}

func runConformanceHookWorkflow(t *testing.T, input conformanceHookInput) conformanceHookResult {
	t.Helper()
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		conformanceHookWorkflow,
		workflow.RegisterOptions{Name: "test-conformance-hook"},
	)

	env.ExecuteWorkflow("test-conformance-hook", input)
	require.NoError(t, env.GetWorkflowError())

	var result conformanceHookResult
	require.NoError(t, env.GetWorkflowResult(&result))
	return result
}

func runConformanceHookWorkflowError(t *testing.T, input conformanceHookInput) error {
	t.Helper()
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		conformanceHookWorkflow,
		workflow.RegisterOptions{Name: "test-conformance-hook"},
	)

	env.ExecuteWorkflow("test-conformance-hook", input)
	return env.GetWorkflowError()
}

func conformanceHookWorkflow(
	ctx workflow.Context,
	input conformanceHookInput,
) (conformanceHookResult, error) {
	steps := []StepDefinition{
		{
			StepSpec: StepSpec{
				ID:  "step-1",
				Use: "conformance-check",
				With: StepInputs{
					Payload: map[string]any{
						"check_id": input.CheckID,
					},
				},
			},
		},
	}
	runData := map[string]any{}
	if err := ConformanceCheckSetupHook(ctx, &steps, nil, input.Config, &runData); err != nil {
		return conformanceHookResult{}, err
	}

	return conformanceHookResult{
		Payload: steps[0].With.Payload,
		Config:  steps[0].With.Config,
	}, nil
}
