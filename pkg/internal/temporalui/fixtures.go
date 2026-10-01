// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalui

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FixturesEnv is the directory of a manifest.json harvested from a Temporal UI
// HAR. When set, matching GET /temporal-ui/api/... calls are served from disk
// instead of the upstream UI (HTML/assets still proxy). Dev-only; unset in prod.
const FixturesEnv = "CREDIMI_TEMPORAL_UI_API_FIXTURES"

type fixtureManifest struct {
	RecordedNamespace string         `json:"recordedNamespace"`
	WorkflowID        string         `json:"workflowId"`
	RunID             string         `json:"runId"`
	Entries           []fixtureEntry `json:"entries"`
	Credimi           *struct {
		GetMyWorkflowRun *fixtureEntry `json:"getMyWorkflowRun"`
	} `json:"credimi"`
}

type fixtureEntry struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Query       map[string]string `json:"query"`
	Status      int               `json:"status"`
	ContentType string            `json:"contentType"`
	Body        string            `json:"body"`
	WorkflowID  string            `json:"workflowId"`
	RunID       string            `json:"runId"`
}

type fixtureStore struct {
	dir               string
	recordedNamespace string
	workflowID        string
	runID             string
	entries           []loadedFixture
	myWorkflowRun     *loadedFixture
}

type loadedFixture struct {
	method      string
	pathParts   []string // Path with "{namespace}" left as the literal token
	query       map[string]string
	status      int
	contentType string
	body        []byte
}

var (
	fixtureMu    sync.Mutex
	fixtureCache = map[string]*fixtureStore{}
)

func loadFixtures() (*fixtureStore, error) {
	dir := strings.TrimSpace(os.Getenv(FixturesEnv))
	if dir == "" {
		return nil, nil
	}
	fixtureMu.Lock()
	defer fixtureMu.Unlock()
	if store, ok := fixtureCache[dir]; ok {
		return store, nil
	}
	store, err := openFixtureStore(dir)
	if err != nil {
		return nil, err
	}
	fixtureCache[dir] = store
	return store, nil
}

func openFixtureStore(dir string) (*fixtureStore, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("read Temporal UI API fixtures manifest: %w", err)
	}
	var man fixtureManifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return nil, fmt.Errorf("parse Temporal UI API fixtures manifest: %w", err)
	}
	store := &fixtureStore{
		dir:               dir,
		recordedNamespace: man.RecordedNamespace,
		workflowID:        man.WorkflowID,
		runID:             man.RunID,
		entries:           make([]loadedFixture, 0, len(man.Entries)),
	}
	for i, e := range man.Entries {
		loaded, err := loadFixtureEntry(dir, e, i)
		if err != nil {
			return nil, err
		}
		store.entries = append(store.entries, loaded)
	}
	if man.Credimi != nil && man.Credimi.GetMyWorkflowRun != nil {
		e := *man.Credimi.GetMyWorkflowRun
		if e.WorkflowID == "" {
			e.WorkflowID = man.WorkflowID
		}
		if e.RunID == "" {
			e.RunID = man.RunID
		}
		loaded, err := loadFixtureEntry(dir, e, -1)
		if err != nil {
			return nil, fmt.Errorf("credimi getMyWorkflowRun: %w", err)
		}
		store.workflowID = e.WorkflowID
		store.runID = e.RunID
		store.myWorkflowRun = &loaded
	}
	return store, nil
}

func loadFixtureEntry(dir string, e fixtureEntry, index int) (loadedFixture, error) {
	bodyPath := filepath.Join(dir, e.Body)
	body, err := os.ReadFile(bodyPath)
	if err != nil {
		if index >= 0 {
			return loadedFixture{}, fmt.Errorf("fixture entry %d body %q: %w", index, e.Body, err)
		}
		return loadedFixture{}, fmt.Errorf("body %q: %w", e.Body, err)
	}
	if strings.HasSuffix(e.Body, ".gz") {
		gr, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return loadedFixture{}, fmt.Errorf("gzip open %q: %w", e.Body, err)
		}
		body, err = io.ReadAll(gr)
		_ = gr.Close()
		if err != nil {
			return loadedFixture{}, fmt.Errorf("gzip read %q: %w", e.Body, err)
		}
	}
	method := e.Method
	if method == "" {
		method = http.MethodGet
	}
	status := e.Status
	if status == 0 {
		status = http.StatusOK
	}
	ctype := e.ContentType
	if ctype == "" {
		ctype = "application/json"
	}
	return loadedFixture{
		method:      method,
		pathParts:   strings.Split(strings.Trim(e.Path, "/"), "/"),
		query:       e.Query,
		status:      status,
		contentType: ctype,
		body:        body,
	}, nil
}

// tryServeFixture writes a fixture response when FixturesEnv is set and the
// request matches. ok is false when fixtures are disabled or no entry matches.
func tryServeFixture(w http.ResponseWriter, r *http.Request, namespace string) (ok bool, err error) {
	store, err := loadFixtures()
	if err != nil {
		return false, err
	}
	if store == nil {
		return false, nil
	}
	if !strings.HasPrefix(r.URL.EscapedPath(), PathPrefix+"/api/") {
		return false, nil
	}
	entry, found := store.match(r.Method, r.URL.EscapedPath(), r.URL.Query(), namespace)
	if !found {
		return false, nil
	}
	body := entry.body
	if store.recordedNamespace != "" && store.recordedNamespace != namespace {
		body = bytesReplaceAll(body, store.recordedNamespace, namespace)
	}
	w.Header().Set("Content-Type", entry.contentType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(entry.status)
	if r.Method != http.MethodHead {
		_, err = w.Write(body)
	}
	return true, err
}

// LookupMyWorkflowRunFixture returns the Credimi GET /api/my/workflows/.../runs/...
// body when FixturesEnv is set and the ids match the harvested run. ok is false
// when fixtures are disabled or the ids do not match.
func LookupMyWorkflowRunFixture(workflowID, runID string) (body []byte, status int, ok bool, err error) {
	store, err := loadFixtures()
	if err != nil {
		return nil, 0, false, err
	}
	if store == nil || store.myWorkflowRun == nil {
		return nil, 0, false, nil
	}
	if workflowID != store.workflowID || runID != store.runID {
		return nil, 0, false, nil
	}
	return store.myWorkflowRun.body, store.myWorkflowRun.status, true, nil
}

func (s *fixtureStore) match(
	method, escapedPath string,
	query url.Values,
	namespace string,
) (loadedFixture, bool) {
	if method == http.MethodHead {
		method = http.MethodGet
	}
	reqParts := strings.Split(strings.Trim(escapedPath, "/"), "/")
	for _, e := range s.entries {
		if !strings.EqualFold(e.method, method) {
			continue
		}
		if !pathMatches(e.pathParts, reqParts, namespace) {
			continue
		}
		if !queryMatches(e.query, query) {
			continue
		}
		return e, true
	}
	return loadedFixture{}, false
}

func pathMatches(pattern, request []string, namespace string) bool {
	if len(pattern) != len(request) {
		return false
	}
	for i := range pattern {
		want := pattern[i]
		got, err := url.PathUnescape(request[i])
		if err != nil {
			return false
		}
		if want == "{namespace}" {
			if got != namespace {
				return false
			}
			continue
		}
		wantUnesc, err := url.PathUnescape(want)
		if err != nil {
			wantUnesc = want
		}
		if got != wantUnesc {
			return false
		}
	}
	return true
}

func queryMatches(required map[string]string, got url.Values) bool {
	for key, want := range required {
		values, ok := got[key]
		if !ok || len(values) == 0 || values[0] != want {
			return false
		}
	}
	return true
}

func bytesReplaceAll(body []byte, old, new string) []byte {
	if old == "" || old == new {
		return body
	}
	return []byte(strings.ReplaceAll(string(body), old, new))
}
