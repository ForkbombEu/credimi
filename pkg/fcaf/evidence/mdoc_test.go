// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"encoding/base64"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/require"
)

func TestParseMDocPresentationPreservesElementTypesAndTags(t *testing.T) {
	raw := testMDocDeviceResponse(t, map[string]any{
		"family_name": "Trotter",
		"birth_date":  cbor.Tag{Number: 1004, Content: "1999-11-01"},
		"nationality": []any{"IT", "GR"},
		"sex":         uint64(1),
		"portrait":    []byte{0xff, 0xd8, 0xff},
	})

	presentation, err := ParseMDocPresentation(base64.RawURLEncoding.EncodeToString(raw))

	require.NoError(t, err)
	require.Equal(t, raw, presentation.Raw)
	require.Len(t, presentation.Documents, 1)
	require.Equal(t, pidMDocTestDocType, presentation.Documents[0].DocType)

	familyName, found := presentation.Element(pidMDocTestDocType, "family_name")
	require.True(t, found)
	require.Equal(t, uint8(3), familyName.MajorType)
	require.Equal(t, "Trotter", familyName.Value)

	birthDate, found := presentation.Element(pidMDocTestDocType, "birth_date")
	require.True(t, found)
	require.Equal(t, uint8(6), birthDate.MajorType)
	require.Equal(t, uint8(3), birthDate.ContentMajorType)
	require.NotNil(t, birthDate.Tag)
	require.Equal(t, uint64(1004), *birthDate.Tag)

	portrait, found := presentation.Element(pidMDocTestDocType, "portrait")
	require.True(t, found)
	require.Equal(t, uint8(2), portrait.MajorType)
	require.Equal(t, []byte{0xff, 0xd8, 0xff}, portrait.Value)
}

func TestParseMDocPresentationRejectsDuplicateElements(t *testing.T) {
	item := testIssuerSignedItem(t, "family_name", "Trotter")
	raw := testMDocResponseWithItems(t, []any{item, item})

	_, err := ParseMDocPresentation(raw)

	require.ErrorContains(t, err, `duplicate element "family_name"`)
}

func TestParseMDocPresentationRejectsInvalidTag(t *testing.T) {
	inner, err := cbor.Marshal(map[string]any{
		"elementIdentifier": "family_name",
		"elementValue":      "Trotter",
	})
	require.NoError(t, err)
	raw := testMDocResponseWithItems(t, []any{cbor.Tag{Number: 23, Content: inner}})

	_, err = ParseMDocPresentation(raw)

	require.ErrorContains(t, err, "expected 24")
}

const pidMDocTestDocType = "eu.europa.ec.eudi.pid.1"

func testMDocDeviceResponse(t *testing.T, elements map[string]any) []byte {
	t.Helper()
	items := make([]any, 0, len(elements))
	for identifier, value := range elements {
		items = append(items, testIssuerSignedItem(t, identifier, value))
	}
	return testMDocResponseWithItems(t, items)
}

func testIssuerSignedItem(t *testing.T, identifier string, value any) cbor.Tag {
	t.Helper()
	encodedValue, err := cbor.Marshal(value)
	require.NoError(t, err)
	inner, err := cbor.Marshal(map[string]any{
		"digestID":          uint64(1),
		"random":            []byte("salt"),
		"elementIdentifier": identifier,
		"elementValue":      cbor.RawMessage(encodedValue),
	})
	require.NoError(t, err)
	return cbor.Tag{Number: 24, Content: inner}
}

func testMDocResponseWithItems(t *testing.T, items []any) []byte {
	t.Helper()
	raw, err := cbor.Marshal(map[string]any{
		"version": "1.0",
		"documents": []any{
			map[string]any{
				"docType": pidMDocTestDocType,
				"issuerSigned": map[string]any{
					"nameSpaces": map[string]any{
						pidMDocTestDocType: items,
					},
				},
			},
		},
		"status": uint64(0),
	})
	require.NoError(t, err)
	return raw
}
