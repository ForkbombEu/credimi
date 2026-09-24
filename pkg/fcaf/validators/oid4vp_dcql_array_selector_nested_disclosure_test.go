// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// nestedDisclosure mirrors one Capture Wallet degree disclosure: a three-element
// object disclosure, or a two-element array-element disclosure.
type nestedDisclosure struct {
	raw    string
	digest string
}

func newNestedDisclosure(t *testing.T, parts ...any) nestedDisclosure {
	t.Helper()
	encoded, err := json.Marshal(parts)
	require.NoError(t, err)
	raw := base64.RawURLEncoding.EncodeToString(encoded)
	digest := sha256.Sum256([]byte(raw))
	return nestedDisclosure{raw: raw, digest: base64.RawURLEncoding.EncodeToString(digest[:])}
}

func arrayPlaceholders(disclosures ...nestedDisclosure) []any {
	placeholders := make([]any, 0, len(disclosures))
	for _, disclosure := range disclosures {
		placeholders = append(placeholders, map[string]any{"...": disclosure.digest})
	}
	return placeholders
}

func nestedDisclosurePresentation(
	t *testing.T,
	claimDisclosure nestedDisclosure,
	presented ...nestedDisclosure,
) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "none"})
	require.NoError(t, err)
	payload, err := json.Marshal(map[string]any{
		"vct":     "urn:credimi:degree:1",
		"_sd_alg": "sha-256",
		"_sd":     []any{claimDisclosure.digest},
	})
	require.NoError(t, err)
	token := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".signature"
	segments := make([]string, 0, len(presented)+1)
	segments = append(segments, claimDisclosure.raw)
	for _, disclosure := range presented {
		segments = append(segments, disclosure.raw)
	}
	return token + "~" + strings.Join(segments, "~") + "~"
}

func nestedSelectorEvidence(path []any, presentation string) map[string]any {
	return map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{map[string]any{
				"id":     "degree",
				"format": "dc+sd-jwt",
				"meta":   map[string]any{"vct_values": []any{"urn:credimi:degree:1"}},
				"claims": []any{map[string]any{"path": path}},
			}},
		},
		"vp_token": map[string]any{"degree": []any{presentation}},
	}
}

// TestArraySelectorFilterReadsCaptureNestedDegreeDisclosures pins the validator
// against the per-element disclosure frame the Capture Wallet issuer signs:
// each degrees entry is an array-element disclosure whose `type` and
// `university` are separate object disclosures.
func TestArraySelectorFilterReadsCaptureNestedDegreeDisclosures(t *testing.T) {
	firstType := newNestedDisclosure(t, "salt-type-0", "type", "Bachelor of Science")
	secondType := newNestedDisclosure(t, "salt-type-1", "type", "Master of Science")
	firstUniversity := newNestedDisclosure(
		t,
		"salt-university-0",
		"university",
		"University of Betelgeuse",
	)
	secondUniversity := newNestedDisclosure(
		t,
		"salt-university-1",
		"university",
		"University of Betelgeuse",
	)
	thirdUniversity := newNestedDisclosure(
		t,
		"salt-university-2",
		"university",
		"University of Betelgeuse",
	)
	firstEntry := newNestedDisclosure(t, "salt-entry-0", map[string]any{
		"_sd": []any{firstType.digest, firstUniversity.digest},
	})
	secondEntry := newNestedDisclosure(t, "salt-entry-1", map[string]any{
		"_sd": []any{secondType.digest, secondUniversity.digest},
	})
	thirdEntry := newNestedDisclosure(t, "salt-entry-2", map[string]any{
		"_sd": []any{thirdUniversity.digest},
	})
	degrees := newNestedDisclosure(
		t,
		"salt-degrees",
		"degrees",
		arrayPlaceholders(firstEntry, secondEntry, thirdEntry),
	)

	path := []any{"degrees", nil, "type"}
	params := map[string]any{
		"vct":              "urn:credimi:degree:1",
		"path":             path,
		"required_values":  []any{"Bachelor of Science", "Master of Science"},
		"forbidden_values": []any{"University of Betelgeuse"},
	}
	validator := OID4VPDCQLArraySelectorFilterValidator{}

	filtered := validator.Validate(context.Background(), Input{
		Value: nestedSelectorEvidence(path, nestedDisclosurePresentation(
			t,
			degrees,
			firstEntry, secondEntry, firstType, secondType,
		)),
		Params: params,
	})
	require.Equal(t, StatusPass, filtered.Status, filtered.Message)

	unfiltered := validator.Validate(context.Background(), Input{
		Value: nestedSelectorEvidence(path, nestedDisclosurePresentation(
			t,
			degrees,
			firstEntry, secondEntry, thirdEntry,
			firstType, secondType,
			firstUniversity, secondUniversity, thirdUniversity,
		)),
		Params: params,
	})
	require.Equal(t, StatusFail, unfiltered.Status, unfiltered.Message)
}

// TestArraySelectorFilterReadsCaptureNestedProgrammeDisclosures pins the
// validator against the nested array-of-arrays disclosure frame, where each
// programme string is its own array-element disclosure.
func TestArraySelectorFilterReadsCaptureNestedProgrammeDisclosures(t *testing.T) {
	bachelor := newNestedDisclosure(t, "salt-programme-0-0", "Bachelor of Science")
	master := newNestedDisclosure(t, "salt-programme-1-0", "Master of Science")
	doctor := newNestedDisclosure(t, "salt-programme-1-1", "Doctor of Philosophy")
	firstProgramme := newNestedDisclosure(t, "salt-array-0", arrayPlaceholders(bachelor))
	secondProgramme := newNestedDisclosure(t, "salt-array-1", arrayPlaceholders(master, doctor))
	programmes := newNestedDisclosure(
		t,
		"salt-programmes",
		"academic_programmes",
		arrayPlaceholders(firstProgramme, secondProgramme),
	)

	path := []any{"academic_programmes", nil, 1}
	params := map[string]any{
		"vct":              "urn:credimi:degree:1",
		"path":             path,
		"required_values":  []any{"Doctor of Philosophy"},
		"forbidden_values": []any{"Bachelor of Science", "Master of Science"},
	}
	validator := OID4VPDCQLArraySelectorFilterValidator{}

	filtered := validator.Validate(context.Background(), Input{
		Value: nestedSelectorEvidence(path, nestedDisclosurePresentation(
			t,
			programmes,
			secondProgramme, doctor,
		)),
		Params: params,
	})
	require.Equal(t, StatusPass, filtered.Status, filtered.Message)

	outOfRangeArrayKept := validator.Validate(context.Background(), Input{
		Value: nestedSelectorEvidence(path, nestedDisclosurePresentation(
			t,
			programmes,
			firstProgramme, secondProgramme,
			bachelor, master, doctor,
		)),
		Params: params,
	})
	require.Equal(t, StatusFail, outOfRangeArrayKept.Status, outOfRangeArrayKept.Message)
}
