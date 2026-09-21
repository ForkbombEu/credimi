// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeFixtureTree(t *testing.T, root string) {
	t.Helper()

	standardDir := filepath.Join(root, "openid4vp")
	require.NoError(t, os.MkdirAll(standardDir, 0o755))
	stdYAML, err := yaml.Marshal(map[string]any{
		"uid":  "openid4vp",
		"name": "OpenID4VP",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(standardDir, "standard.yaml"), stdYAML, 0o644))

	skipDir := filepath.Join(root, "fcaf_sources")
	require.NoError(t, os.MkdirAll(skipDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skipDir, "standard.yaml"), stdYAML, 0o644))

	versionDir := filepath.Join(standardDir, "draft-24")
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	verYAML, err := yaml.Marshal(map[string]any{"uid": "draft-24", "name": "Draft 24"})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "version.yaml"), verYAML, 0o644))

	suiteBoth := filepath.Join(versionDir, "ewc")
	require.NoError(t, os.MkdirAll(suiteBoth, 0o755))
	metaBoth, err := yaml.Marshal(map[string]any{"uid": "ewc", "name": "EWC"})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(suiteBoth, "metadata.yaml"), metaBoth, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(suiteBoth, "check_one.yaml"), []byte("name: Named Check One\ndescription: x\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(suiteBoth, "check_two.yaml"), []byte("title: Titled Check Two\n"), 0o644))

	suiteManual := filepath.Join(versionDir, "oidf")
	require.NoError(t, os.MkdirAll(suiteManual, 0o755))
	metaManual, err := yaml.Marshal(map[string]any{
		"uid":        "oidf",
		"name":       "OIDF",
		"visible_in": []string{"manual"},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(suiteManual, "metadata.yaml"), metaManual, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(suiteManual, "manual_only.yaml"), []byte("session: {}\n"), 0o644))

	suitePipe := filepath.Join(versionDir, "pipe")
	require.NoError(t, os.MkdirAll(suitePipe, 0o755))
	metaPipe, err := yaml.Marshal(map[string]any{
		"uid":        "pipe",
		"name":       "Pipe",
		"visible_in": []string{"pipeline"},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(suitePipe, "metadata.yaml"), metaPipe, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(suitePipe, "pipe_only.yaml"), []byte("{}\n"), 0o644))
}

func TestPathIDStable(t *testing.T) {
	a := PathID("openid4vp/draft-24/ewc/check_one")
	b := PathID("openid4vp/draft-24/ewc/check_one")
	c := PathID("openid4vp/draft-24/ewc/check_two")
	require.Equal(t, a, b)
	require.NotEqual(t, a, c)
	require.Len(t, a, 15)
}

func TestLoadWalkTitlesAndVisibility(t *testing.T) {
	root := t.TempDir()
	writeFixtureTree(t, root)

	checks, err := LoadWalk(root)
	require.NoError(t, err)
	require.Len(t, checks, 4)

	byPath := map[string]Check{}
	for _, ch := range checks {
		byPath[ch.Path] = ch
		require.Equal(t, PathID(ch.Path), ch.ID)
		require.Empty(t, ch.Protocol)
		require.Empty(t, ch.SUT)
		require.Empty(t, ch.Role)
		require.Empty(t, ch.Provider)
	}

	require.Equal(t, "Named Check One", byPath["openid4vp/draft-24/ewc/check_one"].Title)
	require.Equal(t, "Titled Check Two", byPath["openid4vp/draft-24/ewc/check_two"].Title)
	require.Equal(t, "manual_only", byPath["openid4vp/draft-24/oidf/manual_only"].Title)
	require.Equal(t, []string{SurfaceManual}, byPath["openid4vp/draft-24/oidf/manual_only"].VisibleIn)
	require.Equal(t, []string{SurfacePipeline}, byPath["openid4vp/draft-24/pipe/pipe_only"].VisibleIn)
	require.ElementsMatch(t, []string{SurfaceManual, SurfacePipeline}, byPath["openid4vp/draft-24/ewc/check_one"].VisibleIn)

	for _, ch := range checks {
		require.NotEqual(t, "fcaf_sources", ch.Standard)
	}
}

func TestRebuildProjectsIntoCollection(t *testing.T) {
	root := t.TempDir()
	writeFixtureTree(t, root)

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	Register(app)
	require.NoError(t, Rebuild(app, root))

	records, err := app.FindAllRecords(CollectionName)
	require.NoError(t, err)
	require.Len(t, records, 4)

	snap := Default().Snapshot()
	require.Len(t, snap, 4)

	one := byPathRecord(t, records, "openid4vp/draft-24/ewc/check_one")
	require.Equal(t, "Named Check One", one.GetString("title"))
	require.Equal(t, PathID("openid4vp/draft-24/ewc/check_one"), one.Id)

	require.NoError(t, os.WriteFile(
		filepath.Join(root, "openid4vp", "draft-24", "ewc", "check_three.yaml"),
		[]byte("name: Third\n"),
		0o644,
	))
	require.NoError(t, Rebuild(app, root))
	records, err = app.FindAllRecords(CollectionName)
	require.NoError(t, err)
	require.Len(t, records, 5)
}

func TestCollectionListGetFilterAndWriteRejection(t *testing.T) {
	root := t.TempDir()
	writeFixtureTree(t, root)

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	Register(app)
	require.NoError(t, Rebuild(app, root))

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)
	var mux http.Handler
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	require.NoError(t, app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		built, buildErr := e.Router.BuildMux()
		require.NoError(t, buildErr)
		mux = built
		return nil
	}))

	serve := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		var req *http.Request
		if body == "" {
			req = httptest.NewRequest(method, path, nil)
		} else {
			req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
		}
		mux.ServeHTTP(rec, req)
		return rec
	}

	rec := serve(http.MethodGet, "/api/collections/conformance_checks/records?perPage=2&page=1", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	body := rec.Body.String()
	require.Contains(t, body, `"page":1`)
	require.Contains(t, body, `"perPage":2`)
	require.Contains(t, body, `"totalItems":4`)
	require.Contains(t, body, `"items":`)

	rec = serve(http.MethodGet,
		"/api/collections/conformance_checks/records?filter="+url.QueryEscape(`standard="openid4vp"`),
		"")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"totalItems":4`)

	manualRecords, err := app.FindRecordsByFilter(CollectionName, `visible_in ~ {:surface}`, "-path", 0, 0, map[string]any{"surface": SurfaceManual})
	require.NoError(t, err, "FindRecordsByFilter visible_in")
	require.Len(t, manualRecords, 3)

	rec = serve(http.MethodGet,
		"/api/collections/conformance_checks/records?filter="+url.QueryEscape(`visible_in ~ "manual"`)+"&sort=-path",
		"")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"totalItems":3`)

	id := PathID("openid4vp/draft-24/ewc/check_one")
	rec = serve(http.MethodGet, "/api/collections/conformance_checks/records/"+id, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"path":"openid4vp/draft-24/ewc/check_one"`)
	require.Contains(t, rec.Body.String(), `"title":"Named Check One"`)

	rec = serve(http.MethodGet, "/api/collections/conformance_checks/records/zzzzzzzzzzzzzzz", "")
	require.Equal(t, http.StatusNotFound, rec.Code)

	rec = serve(http.MethodPost, "/api/collections/conformance_checks/records",
		`{"path":"x","title":"y","standard":"a","version":"b","suite":"c","file":"d.yaml"}`)
	require.True(t, rec.Code == http.StatusForbidden || rec.Code == http.StatusBadRequest,
		"create status=%d body=%s", rec.Code, rec.Body.String())

	rec = serve(http.MethodPatch, "/api/collections/conformance_checks/records/"+id,
		`{"title":"hijacked"}`)
	require.True(t, rec.Code == http.StatusForbidden || rec.Code == http.StatusBadRequest,
		"update status=%d body=%s", rec.Code, rec.Body.String())

	rec = serve(http.MethodDelete, "/api/collections/conformance_checks/records/"+id, "")
	require.True(t, rec.Code == http.StatusForbidden || rec.Code == http.StatusBadRequest,
		"delete status=%d body=%s", rec.Code, rec.Body.String())

	collection, err := app.FindCollectionByNameOrId(CollectionName)
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("path", "should/not/save")
	record.Set("title", "nope")
	record.Set("standard", "s")
	record.Set("version", "v")
	record.Set("suite", "u")
	record.Set("file", "f.yaml")
	err = app.Save(record)
	require.Error(t, err)
	var apiErr *router.ApiError
	require.ErrorAs(t, err, &apiErr)
}

func byPathRecord(t *testing.T, records []*core.Record, path string) *core.Record {
	t.Helper()
	for _, r := range records {
		if r.GetString("path") == path {
			return r
		}
	}
	t.Fatalf("record with path %q not found", path)
	return nil
}
