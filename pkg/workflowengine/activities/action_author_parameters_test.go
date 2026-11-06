// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
)

// buildSessionLookup fakes a running, Credimi-signed session that authors on
// session's build.
func buildSessionLookup(
	t *testing.T,
	app core.App,
	session actionauthor.BuildRequest,
) sessionLookup {
	t.Helper()
	lookup := validSessionLookup(t, app)
	lookup.history = signedSessionHistory(t, app, map[string]any{
		"platform":                  session.Platform,
		"wallet_version_identifier": session.WalletVersionIdentifier,
		"actor_id":                  session.ActorID,
	})
	return lookup
}

// seedDynamicCredential adds a credential of the session's organization whose
// offer StepCI generates, and returns its path.
func seedDynamicCredential(t *testing.T, app core.App, session actionauthor.BuildRequest) string {
	t.Helper()
	selection, err := actionauthor.Select(app, "", session)
	require.NoError(t, err)
	save := func(collection string, fields map[string]any) *core.Record {
		c, err := app.FindCollectionByNameOrId(collection)
		require.NoError(t, err)
		record := core.NewRecord(c)
		for key, value := range fields {
			record.Set(key, value)
		}
		require.NoError(t, app.Save(record))
		return record
	}
	issuer := save("credential_issuers", map[string]any{
		"owner": selection.Organization.Id,
		"name":  "Authoring issuer",
		"url":   "https://issuer.example.test",
	})
	credential := save("credentials", map[string]any{
		"owner":             selection.Organization.Id,
		"credential_issuer": issuer.Id,
		"name":              "Authoring PID",
		"yaml":              "version: \"1.1\"\n",
	})
	return selection.Organization.GetString("canonified_name") + "/" +
		issuer.GetString("canonified_name") + "/" + credential.GetString("canonified_name")
}

func TestGenerateActionAuthorParameters(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	credential := seedDynamicCredential(t, app, session)
	var sessions atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/broken":
			w.WriteHeader(http.StatusBadGateway)
			return
		case "/large":
			_, _ = w.Write([]byte(strings.Repeat("x", actionAuthorSourceBodyLimit+1)))
			return
		}
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "pid", body["credential_configuration_id"])
		n := sessions.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"deeplink": "openid-credential-offer://x-" + strconv.Itoa(int(n)),
			"session":  map[string]any{"id": "s-" + strconv.Itoa(int(n)), "n": n},
		})
	}))
	t.Cleanup(server.Close)
	httpSource := actionauthor.SourceSpec{
		Kind: actionauthor.SourceHTTP,
		Request: &actionauthor.HTTPSourceRequest{
			Method: http.MethodPost,
			URL:    server.URL + "/sessions",
			Body:   map[string]any{"credential_configuration_id": "pid"},
		},
		ValuePath: "deeplink",
		Captures:  map[string]string{"session_id": "session.id", "number": "session.n"},
	}
	getPID := seedAuthorAction(t, app, session, "Get PID",
		"appId: eu.wallet\n---\n- openLink: ${offer}\n")
	spec := actionauthor.Spec{
		Slot: "get-credential",
		Prerequisites: []actionauthor.PrerequisiteSpec{{
			Action: getPID,
			Parameters: []actionauthor.ParameterSpec{
				{
					Name: "offer",
					Source: actionauthor.SourceSpec{
						Kind:       actionauthor.SourceCredentialOffer,
						Identifier: credential,
					},
				},
			},
		}},
		Parameters: []actionauthor.ParameterSpec{
			{Name: "deeplink", Source: httpSource},
			{
				Name:   "PIN",
				Source: actionauthor.SourceSpec{Kind: actionauthor.SourceLiteral, Value: "123456"},
			},
		},
	}
	caller := authorTestCaller()
	caller.Namespace = namespace
	generate := func(
		plan actionauthor.Spec,
		scope string,
		index int,
	) (GenerateActionAuthorParametersOutput, error) {
		a := NewGenerateActionAuthorParametersActivity(app)
		a.authorSessionLookup = buildSessionLookup(t, app, session).lookup()
		a.transport = server.Client().Transport
		a.runStepCI = func(_ context.Context, code string, _ map[string]string) (string, error) {
			require.Equal(t, "version: \"1.1\"\n", code)
			return "openid-credential-offer://dynamic", nil
		}
		return a.generate(t.Context(), caller, GenerateActionAuthorParametersInput{
			Scope: scope,
			Index: index,
			Plan:  plan,
		})
	}

	first, err := generate(spec, ParameterScopeCandidate, 0)
	require.NoError(t, err)
	require.Equal(t, GenerateActionAuthorParametersOutput{
		Values: map[string]string{"deeplink": "openid-credential-offer://x-1", "PIN": "123456"},
		Captures: map[string]map[string]string{
			"deeplink": {"session_id": "s-1", "number": "1"},
		},
	}, first)
	second, err := generate(spec, ParameterScopeCandidate, 0)
	require.NoError(t, err)
	require.Equal(t, "openid-credential-offer://x-2", second.Values["deeplink"])

	prerequisite, err := generate(spec, ParameterScopePrerequisite, 0)
	require.NoError(t, err)
	require.Equal(
		t,
		map[string]string{"offer": "openid-credential-offer://dynamic"},
		prerequisite.Values,
	)

	_, err = generate(spec, ParameterScopePrerequisite, 1)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "the session has no prerequisite 1")

	broken := spec
	broken.Parameters = []actionauthor.ParameterSpec{{Name: "deeplink", Source: httpSource}}
	broken.Parameters[0].Source.Request = &actionauthor.HTTPSourceRequest{
		Method: http.MethodGet,
		URL:    server.URL + "/broken",
	}
	_, err = generate(broken, ParameterScopeCandidate, 0)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "parameter deeplink: GET "+server.URL+"/broken answered HTTP 502")

	large := spec
	large.Parameters = []actionauthor.ParameterSpec{{Name: "deeplink", Source: httpSource}}
	large.Parameters[0].Source.Request = &actionauthor.HTTPSourceRequest{
		Method: http.MethodGet,
		URL:    server.URL + "/large",
	}
	_, err = generate(large, ParameterScopeCandidate, 0)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "response body exceeds 1048576 bytes")

	missing := spec
	missing.Parameters = []actionauthor.ParameterSpec{{Name: "deeplink", Source: httpSource}}
	missing.Parameters[0].Source.ValuePath = "session.missing"
	_, err = generate(missing, ParameterScopeCandidate, 0)
	require.ErrorContains(t, err, "parameter deeplink: session.missing is not in the response")

	foreign := spec
	foreign.Parameters = []actionauthor.ParameterSpec{{
		Name: "offer",
		Source: actionauthor.SourceSpec{
			Kind:       actionauthor.SourceCredentialOffer,
			Identifier: "userb-s-organization/issuer/missing",
		},
	}}
	_, err = generate(foreign, ParameterScopeCandidate, 0)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "userb-s-organization/issuer/missing not found")
}

func TestSourceRedirectPolicy(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hop, _ := strconv.Atoi(r.URL.Query().Get("hop"))
		if hop > 0 {
			http.Redirect(w, r, "/?hop="+strconv.Itoa(hop-1), http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(plain.Close)
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/", http.StatusFound)
	}))
	t.Cleanup(secure.Close)
	client := &http.Client{Transport: sourceRedirectPolicy{next: secure.Client().Transport}}
	get := func(rawURL string) error {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		return err
	}

	require.NoError(t, get(plain.URL+"/?hop=5"))
	require.ErrorContains(t, get(plain.URL+"/?hop=6"), "stopped after 5 redirects")
	require.ErrorContains(t, get(secure.URL+"/"), "refused redirect from https to http")
}

func TestGenerateActionAuthorParametersRejectsCallersOutsideASession(t *testing.T) {
	setFCAFTestEnv(t)
	lookup := validSessionLookup(t, nil)
	lookup.author = describeExecution(
		ActionAuthorWorkflowType,
		enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		&commonpb.WorkflowExecution{WorkflowId: "ActionAuthorSession-other", RunId: "run"},
	)
	a := NewGenerateActionAuthorParametersActivity(nil)
	a.authorSessionLookup = lookup.lookup()

	_, err := a.generate(t.Context(), authorTestCaller(), GenerateActionAuthorParametersInput{
		Scope: ParameterScopeCandidate,
	})

	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "caller is not the child of its action author session")

	_, err = executeActivity(
		t,
		a,
		GenerateActionAuthorParametersInput{Scope: ParameterScopeCandidate},
	)
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "caller is not an action author session agent")
}

func TestGenerateActionAuthorParametersRejectsSessionsCredimiDidNotSign(t *testing.T) {
	setFCAFTestEnv(t)
	app := newCredimiTestApp(t)
	session := authorTestSession()
	for name, history := range map[string]*fcafHistoryIterator{
		"unsigned": sessionHistory(t, map[string]any{
			"device_id":                 "tenant/runner-1/device-1",
			"platform":                  session.Platform,
			"wallet_version_identifier": session.WalletVersionIdentifier,
			"actor_id":                  session.ActorID,
		}),
		"grant for another session": grantForAnotherSession(t, app),
	} {
		t.Run(name, func(t *testing.T) {
			lookup := validSessionLookup(t, app)
			lookup.history = history
			a := NewGenerateActionAuthorParametersActivity(app)
			a.authorSessionLookup = lookup.lookup()

			_, err := a.generate(
				t.Context(),
				authorTestCaller(),
				GenerateActionAuthorParametersInput{
					Scope: ParameterScopeCandidate,
				},
			)

			requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
			require.ErrorContains(t, err, "action author session was not started by Credimi")
		})
	}
}

func TestCheckActionAuthorProtocolOutcome(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	getPID := seedAuthorAction(t, app, session, "Get PID", "appId: eu.wallet\n---\n- launchApp\n")
	caller := authorTestCaller()
	caller.Namespace = namespace
	var status atomic.Value
	var mu sync.Mutex
	var paths []string
	requested := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), paths...)
	}
	reset := func() {
		mu.Lock()
		defer mu.Unlock()
		paths = nil
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.EscapedPath())
		mu.Unlock()
		_ = json.NewEncoder(w).
			Encode(map[string]any{"result": map[string]any{"status": status.Load()}})
	}))
	t.Cleanup(server.Close)
	spec := func(outcome string) actionauthor.Spec {
		return actionauthor.Spec{
			Slot:          "verify-credential",
			Outcome:       outcome,
			Prerequisites: []actionauthor.PrerequisiteSpec{{Action: getPID}},
			Parameters: []actionauthor.ParameterSpec{{
				Name: "deeplink",
				Source: actionauthor.SourceSpec{
					Kind: actionauthor.SourceHTTP,
					Request: &actionauthor.HTTPSourceRequest{
						Method: http.MethodPost,
						URL:    server.URL + "/sessions",
					},
					ValuePath: "deeplink",
					Captures:  map[string]string{"session_id": "id"},
					Status: &actionauthor.StatusSpec{
						URL:       server.URL + "/sessions/${session_id}",
						Path:      "result.status",
						Success:   []string{"presentation_validated"},
						Rejection: []string{"request_retrieved"},
					},
				},
			}},
		}
	}
	check := func(
		outcome, current string,
		captures map[string]string,
	) (CheckActionAuthorProtocolOutcomeOutput, error) {
		status.Store(current)
		a := NewCheckActionAuthorProtocolOutcomeActivity(app)
		a.authorSessionLookup = buildSessionLookup(t, app, session).lookup()
		a.transport = server.Client().Transport
		a.interval = 5 * time.Millisecond
		a.timeout = 30 * time.Millisecond
		return a.check(t.Context(), caller, CheckActionAuthorProtocolOutcomeInput{
			Parameter: "deeplink",
			Captures:  captures,
			Plan:      spec(outcome),
		})
	}
	captures := map[string]string{"session_id": "a/b c"}

	reached, err := check(actionauthor.OutcomeSuccess, "presentation_validated", captures)
	require.NoError(t, err)
	require.Equal(t, CheckActionAuthorProtocolOutcomeOutput{
		Reached:  true,
		Status:   "presentation_validated",
		Expected: []string{"presentation_validated"},
	}, reached)
	require.Equal(t, "/sessions/a%2Fb%20c", requested()[0])

	reset()
	opposite, err := check(actionauthor.OutcomeRejection, "presentation_validated", captures)
	require.NoError(t, err)
	require.Equal(t, CheckActionAuthorProtocolOutcomeOutput{
		Status:   "presentation_validated",
		Expected: []string{"request_retrieved"},
	}, opposite)
	require.Len(t, requested(), 1, "a status in the opposite list ends the polling")

	reset()
	pending, err := check(actionauthor.OutcomeSuccess, "pending", captures)
	require.NoError(t, err)
	require.False(t, pending.Reached)
	require.Equal(t, "pending", pending.Status)
	require.Greater(t, len(requested()), 1)

	server.Close()
	unreachable, err := check(actionauthor.OutcomeSuccess, "pending", captures)
	require.NoError(t, err)
	require.False(t, unreachable.Reached)
	require.Contains(t, unreachable.Status, "unreachable: ")

	_, err = check(actionauthor.OutcomeSuccess, "pending", map[string]string{})
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "capture session_id is missing")

	a := NewCheckActionAuthorProtocolOutcomeActivity(app)
	a.authorSessionLookup = buildSessionLookup(t, app, session).lookup()
	_, err = a.check(t.Context(), caller, CheckActionAuthorProtocolOutcomeInput{
		Parameter: "PIN",
		Plan:      spec(actionauthor.OutcomeSuccess),
	})
	require.ErrorContains(t, err, "parameter PIN has no protocol status check")
}

func TestStatusURLKeepsTheTemplateHost(t *testing.T) {
	filled, err := statusURL(
		"https://issuer.test/sessions/${id}?x=${id}",
		map[string]string{"id": "@evil.test/a"},
	)
	require.NoError(t, err)
	require.Equal(t, "https://issuer.test/sessions/@evil.test%2Fa?x=@evil.test%2Fa", filled)

	for _, template := range []string{
		"https://issuer.test${id}/sessions",
		"https://${id}/sessions",
	} {
		_, err := statusURL(template, map[string]string{"id": ".evil.test"})
		require.ErrorIs(t, err, actionauthor.ErrInvalid, template)
		require.ErrorContains(t, err, "the filled status url must keep the host")
	}
}
