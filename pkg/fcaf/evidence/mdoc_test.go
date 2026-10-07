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
	require.Equal(t, "SHA-256", presentation.Documents[0].DigestAlgorithm)

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

func TestParseMDocPresentationExposesSecurityObjectStatus(t *testing.T) {
	item := testIssuerSignedItem(t, "family_name", "Trotter")
	raw := testMDocResponseWithDocuments(t, []any{
		testMDocDocument(pidMDocTestDocType, []any{item}, testMDocIssuerAuth(t, map[string]any{
			"digestAlgorithm": "SHA-256",
			"status": map[string]any{
				"status_list": map[string]any{
					"idx": uint64(412),
					"uri": cbor.Tag{Number: 32, Content: "https://issuer.test/status/1"},
				},
			},
		})),
	})

	presentation, err := ParseMDocPresentation(raw)
	require.NoError(t, err)

	status, found := presentation.SecurityObjectStatus()
	require.True(t, found)
	require.Equal(t, uint8(5), status.MajorType)
	require.Nil(t, status.Tag)

	index, found := status.Member("status_list", "idx")
	require.True(t, found)
	require.Equal(t, uint8(0), index.MajorType)
	require.Equal(t, uint64(412), index.Value)

	uri, found := status.Member("status_list", "uri")
	require.True(t, found)
	require.Equal(t, uint8(6), uri.MajorType)
	require.Equal(t, uint8(3), uri.ContentMajorType)
	require.NotNil(t, uri.Tag)
	require.Equal(t, uint64(32), *uri.Tag)
	require.Equal(t, "https://issuer.test/status/1", uri.Value)

	self, found := status.Member()
	require.True(t, found)
	require.Same(t, status, self)

	_, found = status.Member("status_list", "missing")
	require.False(t, found)
	_, found = status.Member("status_list", "idx", "deeper")
	require.False(t, found)
	_, found = (*MDocCBORValue)(nil).Member("status_list")
	require.False(t, found)
}

func TestMDocPresentationSecurityObjectStatusAbsent(t *testing.T) {
	raw := testMDocDeviceResponse(t, map[string]any{"family_name": "Trotter"})
	presentation, err := ParseMDocPresentation(raw)
	require.NoError(t, err)

	status, found := presentation.SecurityObjectStatus()
	require.False(t, found)
	require.Nil(t, status)

	presentation.SelectedDocument = len(presentation.Documents)
	_, found = presentation.SecurityObjectStatus()
	require.False(t, found)

	_, found = (*MDocPresentation)(nil).SecurityObjectStatus()
	require.False(t, found)
}

func TestMDocPresentationDocumentSelectsByDocType(t *testing.T) {
	const mdlDocType = "org.iso.18013.5.1.mDL"
	raw := testMDocResponseWithDocuments(t, []any{
		testMDocDocument(
			pidMDocTestDocType,
			[]any{testIssuerSignedItem(t, "family_name", "Trotter")},
			testMDocIssuerAuth(t, map[string]any{"digestAlgorithm": "SHA-256"}),
		),
		testMDocDocument(
			mdlDocType,
			[]any{testIssuerSignedItem(t, "family_name", "Doe")},
			testMDocIssuerAuth(t, map[string]any{"digestAlgorithm": "SHA-384"}),
		),
	})

	presentation, err := ParseMDocPresentation(raw)
	require.NoError(t, err)
	require.Len(t, presentation.Documents, 2)

	document, found := presentation.Document(mdlDocType)
	require.True(t, found)
	require.Equal(t, "SHA-384", document.DigestAlgorithm)
	require.Same(t, &presentation.Documents[1], document)
	require.Equal(t, "Doe", document.Namespaces[mdlDocType]["family_name"].Value)

	// The presentation-level namespaces always reflect the first document.
	element, found := presentation.Element(pidMDocTestDocType, "family_name")
	require.True(t, found)
	require.Equal(t, "Trotter", element.Value)
	_, found = presentation.Element(mdlDocType, "family_name")
	require.False(t, found)
	_, found = presentation.Element(pidMDocTestDocType, "given_name")
	require.False(t, found)

	_, found = presentation.Document("unknown.doctype")
	require.False(t, found)
	_, found = (*MDocPresentation)(nil).Document(pidMDocTestDocType)
	require.False(t, found)
	_, found = (*MDocPresentation)(nil).Element(pidMDocTestDocType, "family_name")
	require.False(t, found)
}

func TestParseMDocPresentationKeepsResponseStatusAndErrors(t *testing.T) {
	issuerAuth := testMDocIssuerAuth(t, map[string]any{"digestAlgorithm": "SHA-256"})
	document := testMDocDocument(
		pidMDocTestDocType,
		[]any{testIssuerSignedItem(t, "family_name", "Trotter")},
		issuerAuth,
	)
	document["errors"] = map[string]any{
		pidMDocTestDocType: map[string]any{"birth_date": int64(1)},
	}
	raw, err := cbor.Marshal(map[string]any{
		"version":        "1.0",
		"documents":      []any{document},
		"documentErrors": []any{map[string]any{"org.iso.18013.5.1.mDL": int64(10)}},
		"status":         uint64(20),
	})
	require.NoError(t, err)

	presentation, err := ParseMDocPresentation(raw)

	require.NoError(t, err)
	require.Equal(t, uint64(20), presentation.Status)
	require.Equal(
		t,
		[]map[string]int64{{"org.iso.18013.5.1.mDL": 10}},
		presentation.DocumentErrors,
	)
	require.Equal(
		t,
		map[string]map[string]int64{pidMDocTestDocType: {"birth_date": 1}},
		presentation.Documents[0].Errors,
	)
	require.Equal(t, "1.0", presentation.DecodedTopLevel["version"])
}

func TestParseMDocPresentationAcceptsUntaggedSign1AndUnwrappedMSO(t *testing.T) {
	mso, err := cbor.Marshal(map[string]any{"digestAlgorithm": "SHA-512"})
	require.NoError(t, err)
	issuerAuth := []any{[]byte{}, map[string]any{}, mso, []byte("signature")}
	raw := testMDocResponseWithDocuments(t, []any{
		testMDocDocument(
			pidMDocTestDocType,
			[]any{testIssuerSignedItem(t, "family_name", "Trotter")},
			issuerAuth,
		),
	})

	presentation, err := ParseMDocPresentation(raw)

	require.NoError(t, err)
	require.Equal(t, "SHA-512", presentation.Documents[0].DigestAlgorithm)
}

func TestParseMDocPresentationWithoutIssuerAuthHasNoDigestAlgorithm(t *testing.T) {
	document := map[string]any{
		"docType": pidMDocTestDocType,
		"issuerSigned": map[string]any{
			"nameSpaces": map[string]any{
				pidMDocTestDocType: []any{testIssuerSignedItem(t, "family_name", "Trotter")},
			},
		},
	}
	raw := testMDocResponseWithDocuments(t, []any{document})

	presentation, err := ParseMDocPresentation(raw)

	require.NoError(t, err)
	require.Empty(t, presentation.Documents[0].DigestAlgorithm)
	require.Nil(t, presentation.Documents[0].MSOStatus)
}

func TestParseMDocPresentationRejectsMalformedResponses(t *testing.T) {
	validItem := testIssuerSignedItem(t, "family_name", "Trotter")
	validAuth := testMDocIssuerAuth(t, map[string]any{"digestAlgorithm": "SHA-256"})
	marshal := func(value any) []byte {
		encoded, err := cbor.Marshal(value)
		require.NoError(t, err)
		return encoded
	}
	withoutIdentifier := cbor.Tag{Number: 24, Content: marshal(map[string]any{
		"elementValue": "Trotter",
	})}
	msoBytes := marshal(marshal(map[string]any{"version": "1.0"}))

	tests := []struct {
		name    string
		raw     []byte
		wantErr string
	}{
		{
			name:    "not CBOR",
			raw:     []byte{0xff},
			wantErr: "decode mdoc DeviceResponse",
		},
		{
			name:    "no documents",
			raw:     marshal(map[string]any{"version": "1.0", "documents": []any{}}),
			wantErr: "contains no documents",
		},
		{
			name: "document without docType",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument("", []any{validItem}, validAuth),
			}),
			wantErr: "mdoc document 0 has no docType",
		},
		{
			name: "item not tagged",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument(pidMDocTestDocType, []any{"plain"}, validAuth),
			}),
			wantErr: "issuer-signed item is not tagged CBOR",
		},
		{
			name: "tag 24 content not bytes",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument(
					pidMDocTestDocType,
					[]any{cbor.Tag{Number: 24, Content: "not-bytes"}},
					validAuth,
				),
			}),
			wantErr: "tag 24 content is not a byte string",
		},
		{
			name: "item without elementIdentifier",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument(pidMDocTestDocType, []any{withoutIdentifier}, validAuth),
			}),
			wantErr: "contains item without elementIdentifier",
		},
		{
			name: "item is not a map",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument(
					pidMDocTestDocType,
					[]any{cbor.Tag{Number: 24, Content: marshal("scalar")}},
					validAuth,
				),
			}),
			wantErr: "issuer-signed item:",
		},
		{
			name: "COSE_Sign1 with wrong arity",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument(pidMDocTestDocType, []any{validItem}, cbor.Tag{
					Number:  18,
					Content: []any{[]byte{}, map[string]any{}, msoBytes},
				}),
			}),
			wantErr: "COSE_Sign1 has 3 entries, expected 4",
		},
		{
			name: "COSE_Sign1 not an array",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument(pidMDocTestDocType, []any{validItem}, "not-sign1"),
			}),
			wantErr: "decode COSE_Sign1",
		},
		{
			name: "COSE_Sign1 payload not bytes",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument(pidMDocTestDocType, []any{validItem}, []any{
					[]byte{}, map[string]any{}, "payload", []byte("signature"),
				}),
			}),
			wantErr: "decode COSE_Sign1 payload",
		},
		{
			name: "MSO without digestAlgorithm",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument(pidMDocTestDocType, []any{validItem}, cbor.Tag{
					Number: 18,
					Content: []any{
						[]byte{}, map[string]any{}, msoBytes, []byte("signature"),
					},
				}),
			}),
			wantErr: "mobile security object has no digestAlgorithm",
		},
		{
			name: "MSO not a map",
			raw: testMDocResponseWithDocuments(t, []any{
				testMDocDocument(pidMDocTestDocType, []any{validItem}, cbor.Tag{
					Number: 18,
					Content: []any{
						[]byte{}, map[string]any{}, marshal(marshal("mso")), []byte("signature"),
					},
				}),
			}),
			wantErr: "decode Mobile Security Object",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			presentation, err := ParseMDocPresentation(tc.raw)
			require.ErrorContains(t, err, tc.wantErr)
			require.Nil(t, presentation)
		})
	}
}

func TestParseMDocPresentationDecodesEncodings(t *testing.T) {
	raw := testMDocDeviceResponse(t, map[string]any{"family_name": "Trotter"})

	tests := []struct {
		name    string
		encoded any
		wantErr string
	}{
		{name: "raw bytes", encoded: raw},
		{name: "base64url unpadded", encoded: base64.RawURLEncoding.EncodeToString(raw)},
		{name: "base64url padded", encoded: base64.URLEncoding.EncodeToString(raw)},
		{name: "base64 unpadded", encoded: base64.RawStdEncoding.EncodeToString(raw)},
		{name: "base64 padded", encoded: base64.StdEncoding.EncodeToString(raw)},
		{
			name:    "not base64",
			encoded: "not base64!",
			wantErr: "mdoc value is not valid base64 or base64url",
		},
		{
			name:    "unsupported type",
			encoded: map[string]any{},
			wantErr: "mdoc value is map[string]interface {}, expected string or bytes",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			presentation, err := ParseMDocPresentation(tc.encoded)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, raw, presentation.Raw)
			element, found := presentation.Element(pidMDocTestDocType, "family_name")
			require.True(t, found)
			require.Equal(t, "Trotter", element.Value)
		})
	}
}

func TestParseMDocPresentationDoesNotAliasCallerBytes(t *testing.T) {
	raw := testMDocDeviceResponse(t, map[string]any{"family_name": "Trotter"})
	original := append([]byte(nil), raw...)

	presentation, err := ParseMDocPresentation(raw)
	require.NoError(t, err)
	for index := range raw {
		raw[index] = 0
	}

	require.Equal(t, original, presentation.Raw)
}

// A map with non-text keys is valid CBOR with no JSON shape: it must not reject
// the presentation, only lose its decoded Value.
func TestParseMDocPresentationKeepsElementWithNonTextMapKeys(t *testing.T) {
	raw := testMDocDeviceResponse(t, map[string]any{
		"family_name": "Trotter",
		"status":      map[uint64]any{1: "valid"},
	})

	presentation, err := ParseMDocPresentation(raw)

	require.NoError(t, err)
	status, found := presentation.Element(pidMDocTestDocType, "status")
	require.True(t, found)
	require.Equal(t, uint8(5), status.MajorType)
	require.Nil(t, status.Value)
	require.NotEmpty(t, status.Raw)
	familyName, found := presentation.Element(pidMDocTestDocType, "family_name")
	require.True(t, found)
	require.Equal(t, "Trotter", familyName.Value)
}

func TestDecodeMDocCBORValueNonTextMapKeys(t *testing.T) {
	raw, err := cbor.Marshal(map[string]any{
		"status_list": map[uint64]any{1: "valid"},
		"uri":         "https://issuer.example/status",
	})
	require.NoError(t, err)

	status, err := decodeMDocCBORValue(raw)

	require.NoError(t, err)
	list, found := status.Member("status_list")
	require.True(t, found)
	require.Equal(t, uint8(5), list.ContentMajorType)
	require.Nil(t, list.Value)
	require.Empty(t, list.Members)
	uri, found := status.Member("uri")
	require.True(t, found)
	require.Equal(t, "https://issuer.example/status", uri.Value)
}

func TestDecodeMDocCBORValueRejectsInvalidUTF8UnderNonTextKey(t *testing.T) {
	// {1: "\xff"}: the non-text key must not hide the invalid text string.
	_, err := decodeMDocCBORValue(cbor.RawMessage{0xa1, 0x01, 0x61, 0xff})

	require.Error(t, err)
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
	return testMDocResponseWithDocuments(t, []any{
		testMDocDocument(
			pidMDocTestDocType,
			items,
			testMDocIssuerAuth(t, map[string]any{"digestAlgorithm": "SHA-256"}),
		),
	})
}

func testMDocDocument(docType string, items []any, issuerAuth any) map[string]any {
	return map[string]any{
		"docType": docType,
		"issuerSigned": map[string]any{
			"nameSpaces": map[string]any{
				docType: items,
			},
			"issuerAuth": issuerAuth,
		},
	}
}

func testMDocIssuerAuth(t *testing.T, mso map[string]any) cbor.Tag {
	t.Helper()
	encoded, err := cbor.Marshal(mso)
	require.NoError(t, err)
	msoBytes, err := cbor.Marshal(encoded)
	require.NoError(t, err)
	return cbor.Tag{Number: 18, Content: []any{
		[]byte{}, map[string]any{}, msoBytes, []byte("signature"),
	}}
}

func testMDocResponseWithDocuments(t *testing.T, documents []any) []byte {
	t.Helper()
	raw, err := cbor.Marshal(map[string]any{
		"version":   "1.0",
		"documents": documents,
		"status":    uint64(0),
	})
	require.NoError(t, err)
	return raw
}
