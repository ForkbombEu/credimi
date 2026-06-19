// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalcrypto_test

import (
	"bytes"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

func TestSecretsDataConverterEncryptsWorkflowInputSecrets(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	dc := temporalcrypto.NewDataConverter(key)

	input := workflowengine.WorkflowInput{
		Payload: map[string]any{"yaml": "steps: []"},
		Config:  map[string]any{"app_url": "https://example.test"},
		Secrets: map[string]any{"token": "super-secret"},
	}

	payload, err := dc.ToPayload(input)
	require.NoError(t, err)
	require.NotContains(t, string(payload.GetData()), "super-secret")
	require.NotContains(t, string(payload.GetData()), `"token"`)
	require.Contains(t, string(payload.GetData()), "__credimi_encrypted_secrets")

	var decoded workflowengine.WorkflowInput
	require.NoError(t, dc.FromPayload(payload, &decoded))
	assert.Equal(t, input.Payload, decoded.Payload)
	assert.Equal(t, input.Config, decoded.Config)
	assert.Equal(t, input.Secrets, decoded.Secrets)
}

func TestSecretsDataConverterDoesNotEncryptPayloadOrConfigSecretsKeys(t *testing.T) {
	key := bytes.Repeat([]byte{8}, 32)
	dc := temporalcrypto.NewDataConverter(key)

	input := workflowengine.WorkflowInput{
		Payload: map[string]any{
			"secrets": map[string]any{"payload_token": "leave-visible"},
		},
		Config: map[string]any{
			"secrets": map[string]any{"config_token": "leave-visible-too"},
		},
		Secrets: map[string]any{"root_token": "encrypt-me"},
	}

	payload, err := dc.ToPayload(input)
	require.NoError(t, err)
	require.Contains(t, string(payload.GetData()), `"payload_token":"leave-visible"`)
	require.Contains(t, string(payload.GetData()), `"config_token":"leave-visible-too"`)
	require.NotContains(t, string(payload.GetData()), `"root_token":"encrypt-me"`)

	var decoded workflowengine.WorkflowInput
	require.NoError(t, dc.FromPayload(payload, &decoded))
	assert.Equal(t, input.Payload, decoded.Payload)
	assert.Equal(t, input.Config, decoded.Config)
	assert.Equal(t, input.Secrets, decoded.Secrets)
}

func TestSecretsDataConverterEncryptsActivityInputSecrets(t *testing.T) {
	key := bytes.Repeat([]byte{2}, 32)
	dc := temporalcrypto.NewDataConverter(key)

	input := workflowengine.ActivityInput{
		Payload: map[string]any{"yaml": "steps: []"},
		Config:  map[string]string{"env": "dev"},
		Secrets: map[string]any{"pin": "1234"},
	}

	payload, err := dc.ToPayload(input)
	require.NoError(t, err)
	require.NotContains(t, string(payload.GetData()), "1234")

	var decoded workflowengine.ActivityInput
	require.NoError(t, dc.FromPayload(payload, &decoded))
	assert.Equal(t, input.Payload, decoded.Payload)
	assert.Equal(t, input.Config, decoded.Config)
	assert.Equal(t, input.Secrets, decoded.Secrets)
}

func TestSecretsDataConverterEncryptsNestedWorkflowInputSecrets(t *testing.T) {
	key := bytes.Repeat([]byte{3}, 32)
	dc := temporalcrypto.NewDataConverter(key)

	input := map[string]any{
		"workflow_input": workflowengine.WorkflowInput{
			Payload: map[string]any{"name": "pipeline"},
			Secrets: map[string]any{"apiKey": "nested-secret"},
		},
	}

	payload, err := dc.ToPayload(input)
	require.NoError(t, err)
	require.NotContains(t, string(payload.GetData()), "nested-secret")

	var decoded map[string]any
	require.NoError(t, dc.FromPayload(payload, &decoded))
	workflowInput, ok := decoded["workflow_input"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, map[string]any{"apiKey": "nested-secret"}, workflowInput["secrets"])
}

func TestSecretsDataConverterDecodesPlaintextSecrets(t *testing.T) {
	key := bytes.Repeat([]byte{4}, 32)
	dc := temporalcrypto.NewDataConverter(key)

	payload := &commonpb.Payload{
		Metadata: map[string][]byte{
			converter.MetadataEncoding: []byte(converter.MetadataEncodingJSON),
		},
		Data: []byte(`{"secrets":{"token":"old-plaintext"},"payload":{"ok":true}}`),
	}

	var decoded workflowengine.WorkflowInput
	require.NoError(t, dc.FromPayload(payload, &decoded))
	assert.Equal(t, map[string]any{"token": "old-plaintext"}, decoded.Secrets)
	assert.Equal(t, map[string]any{"ok": true}, decoded.Payload)
}

func TestSecretsDataConverterWrongKeyFails(t *testing.T) {
	keyA := bytes.Repeat([]byte{5}, 32)
	keyB := bytes.Repeat([]byte{6}, 32)
	encoder := temporalcrypto.NewDataConverter(keyA)
	decoder := temporalcrypto.NewDataConverter(keyB)

	payload, err := encoder.ToPayload(workflowengine.WorkflowInput{
		Secrets: map[string]any{"token": "secret"},
	})
	require.NoError(t, err)

	var decoded workflowengine.WorkflowInput
	require.Error(t, decoder.FromPayload(payload, &decoded))
}

func TestSecretsDataConverterToStringDoesNotDecrypt(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	dc := temporalcrypto.NewDataConverter(key)

	payload, err := dc.ToPayload(workflowengine.WorkflowInput{
		Secrets: map[string]any{"token": "hidden-secret"},
	})
	require.NoError(t, err)

	stringer := temporalcrypto.NewSecretsJSONPayloadConverter(key)
	rendered := stringer.ToString(payload)
	require.NotContains(t, rendered, "hidden-secret")
	require.Contains(t, rendered, "__credimi_encrypted_secrets")
}
