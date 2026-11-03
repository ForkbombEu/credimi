// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func httpSource() SourceSpec {
	return SourceSpec{
		Kind: SourceHTTP,
		Request: &HTTPSourceRequest{
			Method:  "POST",
			URL:     "https://capture-wallet.credimi.io/sessions",
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    map[string]any{"credential_configuration_id": "pid"},
		},
		ValuePath: "deeplink",
		Captures:  map[string]string{"session_id": "session_id"},
		Status: &StatusSpec{
			URL:       "https://capture-wallet.credimi.io/sessions/${session_id}",
			Path:      "status",
			Success:   []string{"done"},
			Rejection: []string{"refused"},
		},
	}
}

func validSpec() Spec {
	return Spec{
		Slot:          "get-credential",
		Prerequisites: []PrerequisiteSpec{{Action: "org/wallet/onboarding"}},
		Parameters:    []ParameterSpec{{Name: "deeplink", Source: httpSource()}},
	}
}

func TestSpecNormalize(t *testing.T) {
	tests := []struct {
		name    string
		edit    func(*Spec)
		message string
	}{
		{name: "valid", edit: func(*Spec) {}},
		{
			name: "onboarding",
			edit: func(s *Spec) { *s = Spec{Slot: "onboarding"} },
		},
		{
			name:    "unknown slot",
			edit:    func(s *Spec) { s.Slot = "install-app" },
			message: "slot: must be one of get-credential, get-credential-specific, onboarding, other, verify-credential",
		},
		{
			name:    "unknown outcome",
			edit:    func(s *Spec) { s.Outcome = "maybe" },
			message: "outcome: must be success or rejection",
		},
		{
			name:    "onboarding rejection",
			edit:    func(s *Spec) { *s = Spec{Slot: "onboarding", Outcome: OutcomeRejection} },
			message: "outcome: the onboarding slot has no rejection outcome",
		},
		{
			name: "onboarding prerequisites",
			edit: func(s *Spec) {
				*s = Spec{Slot: "onboarding", Prerequisites: []PrerequisiteSpec{{Action: "a/b/c"}}}
			},
			message: "prerequisites: the onboarding slot takes no prerequisites",
		},
		{
			name:    "missing prerequisites",
			edit:    func(s *Spec) { s.Prerequisites = nil },
			message: "prerequisites: the get-credential slot needs at least one prerequisite",
		},
		{
			name: "optional prerequisites",
			edit: func(s *Spec) { s.Slot = "other"; s.Prerequisites = nil },
		},
		{
			name: "too many prerequisites",
			edit: func(s *Spec) {
				s.Prerequisites = make([]PrerequisiteSpec, 6)
				for i := range s.Prerequisites {
					s.Prerequisites[i].Action = "a/b/c"
				}
			},
			message: "prerequisites: at most 5",
		},
		{
			name:    "empty prerequisite",
			edit:    func(s *Spec) { s.Prerequisites[0].Action = " " },
			message: "prerequisites[0].action: required",
		},
		{
			name:    "bad parameter name",
			edit:    func(s *Spec) { s.Parameters[0].Name = "1deeplink" },
			message: "parameters[0].name: must match ^[A-Za-z][A-Za-z0-9_]{0,63}$",
		},
		{
			name: "duplicate parameter",
			edit: func(s *Spec) {
				s.Parameters = append(s.Parameters, ParameterSpec{
					Name:   "deeplink",
					Source: SourceSpec{Kind: SourceLiteral, Value: "x"},
				})
			},
			message: "parameters[1].name: duplicate name deeplink",
		},
		{
			name: "too many parameters",
			edit: func(s *Spec) {
				for i := range 8 {
					s.Parameters = append(s.Parameters, ParameterSpec{
						Name:   "p" + strings.Repeat("x", i),
						Source: SourceSpec{Kind: SourceLiteral, Value: "x"},
					})
				}
			},
			message: "parameters: at most 8",
		},
		{
			name: "prerequisite parameter name",
			edit: func(s *Spec) {
				s.Prerequisites[0].Parameters = []ParameterSpec{
					{Name: "", Source: SourceSpec{Kind: SourceLiteral, Value: "x"}},
				}
			},
			message: "prerequisites[0].parameters[0].name: must match ^[A-Za-z][A-Za-z0-9_]{0,63}$",
		},
		{
			name: "credential offer without identifier",
			edit: func(s *Spec) {
				s.Parameters[0].Source = SourceSpec{Kind: SourceCredentialOffer}
			},
			message: "parameters[0].source.identifier: required",
		},
		{
			name: "credential offer with extra field",
			edit: func(s *Spec) {
				s.Parameters[0].Source = SourceSpec{
					Kind:       SourceUseCaseVerification,
					Identifier: "org/verifier/use-case",
					Value:      "x",
				}
			},
			message: "parameters[0].source: kind use_case_verification takes only an identifier",
		},
		{
			name: "valid credential offer",
			edit: func(s *Spec) {
				s.Parameters[0].Source = SourceSpec{
					Kind:       SourceCredentialOffer,
					Identifier: "org/issuer/credential",
				}
			},
		},
		{
			name: "empty literal",
			edit: func(s *Spec) {
				s.Parameters[0].Source = SourceSpec{Kind: SourceLiteral}
			},
			message: "parameters[0].source.value: must be 1-1000 characters",
		},
		{
			name: "long literal",
			edit: func(s *Spec) {
				s.Parameters[0].Source = SourceSpec{
					Kind:  SourceLiteral,
					Value: strings.Repeat("a", 1001),
				}
			},
			message: "parameters[0].source.value: must be 1-1000 characters",
		},
		{
			name:    "unknown kind",
			edit:    func(s *Spec) { s.Parameters[0].Source.Kind = "env" },
			message: "parameters[0].source.kind: must be credential_offer, use_case_verification, http or literal",
		},
		{
			name:    "http without request",
			edit:    func(s *Spec) { s.Parameters[0].Source.Request = nil },
			message: "parameters[0].source.request: required",
		},
		{
			name:    "http method",
			edit:    func(s *Spec) { s.Parameters[0].Source.Request.Method = "PUT" },
			message: "parameters[0].source.request.method: must be GET or POST",
		},
		{
			name:    "http url",
			edit:    func(s *Spec) { s.Parameters[0].Source.Request.URL = "http://example.org" },
			message: "parameters[0].source.request.url: must be an https URL",
		},
		{
			name:    "http url without host",
			edit:    func(s *Spec) { s.Parameters[0].Source.Request.URL = "https://" },
			message: "parameters[0].source.request.url: must be an https URL",
		},
		{
			name: "credential header",
			edit: func(s *Spec) {
				s.Parameters[0].Source.Request.Headers = map[string]string{"X-Api-Key": "k"}
			},
			message: "parameters[0].source.request.headers: only Content-Type, Accept and " +
				"Accept-Language are allowed; credentials must not be stored in Temporal history",
		},
		{
			name: "header outside the allowlist",
			edit: func(s *Spec) {
				s.Parameters[0].Source.Request.Headers = map[string]string{"X-Session": "s"}
			},
			message: "parameters[0].source.request.headers: only Content-Type, Accept and " +
				"Accept-Language are allowed; credentials must not be stored in Temporal history",
		},
		{
			name: "allowed headers, any case",
			edit: func(s *Spec) {
				s.Parameters[0].Source.Request.Headers = map[string]string{
					"content-type":    "application/json",
					"ACCEPT":          "application/json",
					"Accept-Language": "en",
				}
			},
		},
		{
			name: "large body",
			edit: func(s *Spec) {
				s.Parameters[0].Source.Request.Body = strings.Repeat("a", 16*1024)
			},
			message: "parameters[0].source.request.body: must be at most 16 KiB",
		},
		{
			name:    "missing value path",
			edit:    func(s *Spec) { s.Parameters[0].Source.ValuePath = "" },
			message: "parameters[0].source.value_path: required",
		},
		{
			name: "bad capture name",
			edit: func(s *Spec) {
				s.Parameters[0].Source.Captures = map[string]string{"9x": "id"}
				s.Parameters[0].Source.Status = nil
			},
			message: "parameters[0].source.captures.9x: name must match ^[A-Za-z][A-Za-z0-9_]{0,63}$",
		},
		{
			name: "empty capture path",
			edit: func(s *Spec) {
				s.Parameters[0].Source.Captures = map[string]string{"session_id": ""}
			},
			message: "parameters[0].source.captures.session_id: path required",
		},
		{
			name: "status on prerequisite",
			edit: func(s *Spec) {
				s.Prerequisites[0].Parameters = []ParameterSpec{
					{Name: "deeplink", Source: httpSource()},
				}
			},
			message: "prerequisites[0].parameters[0].source.status: only the new action's parameters can check a protocol status",
		},
		{
			name:    "status url",
			edit:    func(s *Spec) { s.Parameters[0].Source.Status.URL = "http://x/${session_id}" },
			message: "parameters[0].source.status.url: must be an https URL",
		},
		{
			name:    "status url unknown capture",
			edit:    func(s *Spec) { s.Parameters[0].Source.Status.URL = "https://x/${other}" },
			message: "parameters[0].source.status.url: ${other} is not a capture",
		},
		{
			name:    "status url host capture",
			edit:    func(s *Spec) { s.Parameters[0].Source.Status.URL = "https://${session_id}/s" },
			message: "parameters[0].source.status.url: the host must not use captures",
		},
		{
			name: "status url host suffix capture",
			edit: func(s *Spec) {
				s.Parameters[0].Source.Status.URL = "https://x.test${session_id}/s"
			},
			message: "parameters[0].source.status.url: the host must not use captures",
		},
		{
			name: "status url port capture",
			edit: func(s *Spec) {
				s.Parameters[0].Source.Status.URL = "https://x.test:${session_id}/s"
			},
			message: "parameters[0].source.status.url: must be an https URL",
		},
		{
			name:    "status path",
			edit:    func(s *Spec) { s.Parameters[0].Source.Status.Path = "" },
			message: "parameters[0].source.status.path: required",
		},
		{
			name:    "status success list",
			edit:    func(s *Spec) { s.Parameters[0].Source.Status.Success = nil },
			message: "parameters[0].source.status.success: required for a success outcome",
		},
		{
			name: "status rejection list",
			edit: func(s *Spec) {
				s.Outcome = OutcomeRejection
				s.Parameters[0].Source.Status.Rejection = nil
			},
			message: "parameters[0].source.status.rejection: required for a rejection outcome",
		},
		{
			name: "rejection outcome",
			edit: func(s *Spec) {
				s.Outcome = OutcomeRejection
				s.Parameters[0].Source.Status.Success = nil
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec := validSpec()
			tc.edit(&spec)
			err := spec.Normalize()
			if tc.message == "" {
				require.NoError(t, err)
				assert.NotEmpty(t, spec.Outcome)
				return
			}
			require.ErrorIs(t, err, ErrInvalid)
			assert.Equal(t, ErrInvalid.Error()+": "+tc.message, err.Error())
		})
	}
}

func TestSpecNormalizeDefaultsOutcome(t *testing.T) {
	spec := Spec{Slot: "onboarding"}
	require.NoError(t, spec.Normalize())
	assert.Equal(t, OutcomeSuccess, spec.Outcome)
}

func TestSlotCategory(t *testing.T) {
	assert.Equal(t, "get-credential-generic", SlotCategory("get-credential"))
	assert.Equal(t, "onboarding", SlotCategory("onboarding"))
	assert.Empty(t, SlotCategory("install-app"))
}

func TestLookupPath(t *testing.T) {
	var body any
	require.NoError(t, json.Unmarshal(
		[]byte(`{"a":{"b":[{"c":"x"},{"n":12.5}],"i":3},"s":"top","t":true}`),
		&body,
	))
	for path, want := range map[string]string{"s": "top", "a.b.0.c": "x", "a.b.1.n": "12.5", "a.i": "3"} {
		got, ok := LookupPath(body, path)
		assert.True(t, ok, path)
		assert.Equal(t, want, got, path)
	}
	for _, path := range []string{"", "missing", "a.b.2", "a.b.x", "t", "a", "s.x"} {
		_, ok := LookupPath(body, path)
		assert.False(t, ok, path)
	}
}

func TestActionParameters(t *testing.T) {
	code := "appId: x\nenv:\n  LOCAL: v\n---\n- openLink: ${deeplink}\n- inputText: ${PIN}\n" +
		"- inputText: ${LOCAL}\n- inputText: ${deeplink}\n- evalScript: ${output.x}\n" +
		"- inputText: ${_private}\n- inputText: ${9lives}\n"
	assert.Equal(t, []string{"PIN", "deeplink"}, ActionParameters(code))
	assert.Equal(t, []string{}, ActionParameters("appId: x\n---\n- launchApp\n"))
}

func TestCheckRecordAccess(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	actions, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	action := core.NewRecord(actions)
	action.Set("owner", selection.Organization.Id)
	action.Set("wallet", selection.Wallet.Id)
	action.Set("name", "Spec onboarding")
	action.Set("code", "appId: x\n---\n- launchApp\n")
	action.Set("category", "onboarding")
	require.NoError(t, app.Save(action))
	path := namespace + "/" + selection.Wallet.GetString("canonified_name") + "/" +
		action.GetString("canonified_name")

	record, err := CheckRecordAccess(app, "wallet_actions", "/"+path, selection.Organization.Id)
	require.NoError(t, err)
	assert.Equal(t, action.Id, record.Id)

	_, err = CheckRecordAccess(app, "wallet_actions", path, "another-org")
	require.ErrorIs(t, err, ErrInvalid)
	assert.Contains(t, err.Error(), path+" not found")

	action.Set("published", true)
	require.NoError(t, app.Save(action))
	_, err = CheckRecordAccess(app, "wallet_actions", path, "another-org")
	require.NoError(t, err)

	_, err = CheckRecordAccess(app, "credentials", path, selection.Organization.Id)
	require.ErrorIs(t, err, ErrInvalid)
	_, err = CheckRecordAccess(
		app,
		"wallet_actions",
		namespace+"/missing/x",
		selection.Organization.Id,
	)
	require.ErrorIs(t, err, ErrInvalid)
}
