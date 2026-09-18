// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/require"
)

const msoStatusTestDocType = "eu.europa.ec.eudi.pid.1"

// msoStatusPresentation builds a device response whose Mobile Security Object
// carries the given `status` value, mirroring the issuer-signed shape the
// Capture Wallet emits when `status_list_enabled` is set.
func msoStatusPresentation(t *testing.T, status any) *evidence.MDocPresentation {
	t.Helper()
	mso := map[string]any{"digestAlgorithm": "SHA-256"}
	if status != nil {
		mso["status"] = status
	}
	encodedMSO, err := cbor.Marshal(mso)
	require.NoError(t, err)
	msoBytes, err := cbor.Marshal(encodedMSO)
	require.NoError(t, err)

	elementValue, err := cbor.Marshal("Arthur")
	require.NoError(t, err)
	item, err := cbor.Marshal(map[string]any{
		"digestID":          uint64(1),
		"random":            []byte("salt"),
		"elementIdentifier": "given_name",
		"elementValue":      cbor.RawMessage(elementValue),
	})
	require.NoError(t, err)

	raw, err := cbor.Marshal(map[string]any{
		"version": "1.0",
		"documents": []any{
			map[string]any{
				"docType": msoStatusTestDocType,
				"issuerSigned": map[string]any{
					"nameSpaces": map[string]any{
						msoStatusTestDocType: []any{cbor.Tag{Number: 24, Content: item}},
					},
					"issuerAuth": cbor.Tag{Number: 18, Content: []any{
						[]byte{}, map[string]any{}, msoBytes, []byte("signature"),
					}},
				},
			},
		},
		"status": uint64(0),
	})
	require.NoError(t, err)

	presentation, err := evidence.ParseMDocPresentation(
		base64.RawURLEncoding.EncodeToString(raw),
	)
	require.NoError(t, err)
	return presentation
}

func tokenStatusList() map[string]any {
	return map[string]any{
		"status_list": map[string]any{
			"idx": uint64(42),
			"uri": "https://beta-capture-wallet.credimi.io/status-lists/1",
		},
	}
}

func TestMSOStatusMemberPresence(t *testing.T) {
	present := MDocMSOStatusMemberPresentValidator{}
	withStatus := msoStatusPresentation(t, tokenStatusList())

	for _, path := range [][]any{
		{},
		{"status_list"},
		{"status_list", "idx"},
		{"status_list", "uri"},
	} {
		result := present.Validate(context.Background(), Input{
			Value:  withStatus,
			Params: map[string]any{"path": path},
		})
		require.Equalf(t, StatusPass, result.Status, "path %v: %s", path, result.Message)
	}

	missingStatus := present.Validate(context.Background(), Input{
		Value:  msoStatusPresentation(t, nil),
		Params: map[string]any{"path": []any{}},
	})
	require.Equal(t, StatusFail, missingStatus.Status, missingStatus.Message)

	identifierListOnly := present.Validate(context.Background(), Input{
		Value: msoStatusPresentation(t, map[string]any{
			"identifier_list": map[string]any{"id": "abc", "uri": "https://example.test/l"},
		}),
		Params: map[string]any{"path": []any{"status_list"}},
	})
	require.Equal(t, StatusFail, identifierListOnly.Status, identifierListOnly.Message)
}

func TestMSOStatusCBORTypes(t *testing.T) {
	cborType := MDocMSOStatusCBORTypeValidator{}
	withStatus := msoStatusPresentation(t, tokenStatusList())

	for _, expectation := range []struct {
		path      []any
		majorType int
	}{
		{path: []any{}, majorType: 5},
		{path: []any{"status_list"}, majorType: 5},
		{path: []any{"status_list", "idx"}, majorType: 0},
		{path: []any{"status_list", "uri"}, majorType: 3},
	} {
		result := cborType.Validate(context.Background(), Input{
			Value: withStatus,
			Params: map[string]any{
				"path":       expectation.path,
				"major_type": expectation.majorType,
			},
		})
		require.Equalf(
			t,
			StatusPass,
			result.Status,
			"path %v: %s",
			expectation.path,
			result.Message,
		)
	}

	// A wallet that re-encoded idx as a text string breaks the CBOR contract.
	rewritten := msoStatusPresentation(t, map[string]any{
		"status_list": map[string]any{
			"idx": "42",
			"uri": "https://beta-capture-wallet.credimi.io/status-lists/1",
		},
	})
	result := cborType.Validate(context.Background(), Input{
		Value:  rewritten,
		Params: map[string]any{"path": []any{"status_list", "idx"}, "major_type": 0},
	})
	require.Equal(t, StatusFail, result.Status, result.Message)

	missingMajorType := cborType.Validate(context.Background(), Input{
		Value:  withStatus,
		Params: map[string]any{"path": []any{}},
	})
	require.Equal(t, StatusError, missingMajorType.Status, missingMajorType.Message)
}

func TestMSOStatusURI(t *testing.T) {
	validator := MDocMSOStatusURIValidator{}
	params := map[string]any{"path": []any{"status_list", "uri"}}

	absolute := validator.Validate(context.Background(), Input{
		Value:  msoStatusPresentation(t, tokenStatusList()),
		Params: params,
	})
	require.Equal(t, StatusPass, absolute.Status, absolute.Message)

	relative := validator.Validate(context.Background(), Input{
		Value: msoStatusPresentation(t, map[string]any{
			"status_list": map[string]any{"idx": uint64(1), "uri": "/status-lists/1"},
		}),
		Params: params,
	})
	require.Equal(t, StatusFail, relative.Status, relative.Message)

	nonString := validator.Validate(context.Background(), Input{
		Value: msoStatusPresentation(t, map[string]any{
			"status_list": map[string]any{"idx": uint64(1), "uri": uint64(7)},
		}),
		Params: params,
	})
	require.Equal(t, StatusFail, nonString.Status, nonString.Message)
}
