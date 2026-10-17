// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validNewsDraft() StoreNewsDraftInput {
	return StoreNewsDraftInput{
		Title:            "Wallet spec v2 released",
		SummaryHTML:      "<p>Summary</p>",
		NewsHTML:         "<p>News body</p>",
		KeyDifferencesMD: "- changed",
		Tags:             []string{"OpenID4VP", "eu-di wallet", "openid4vp", "!!", "", "v2_0"},
		Refer:            "https://example.com/spec",
		Diff:             "https://github.com/example/spec/compare/a...b",
	}
}

func countNews(t *testing.T, app core.App) int64 {
	t.Helper()
	total, err := app.CountRecords("news")
	require.NoError(t, err)
	return total
}

func TestStoreNewsDraftActivityCreatesUnpublishedDraft(t *testing.T) {
	app := newCredimiTestApp(t)
	act := NewStoreNewsDraftActivity(app)

	result, err := executeActivity(t, act, validNewsDraft())
	require.NoError(t, err)
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	id, ok := output["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, id)
	assert.Equal(t, false, output["duplicate"])

	record, err := app.FindRecordById("news", id)
	require.NoError(t, err)
	assert.False(t, record.GetBool("published"))
	assert.Equal(t, "Wallet spec v2 released", record.GetString("title"))
	assert.Equal(t, "<p>Summary</p>", record.GetString("summary"))
	assert.Equal(t, "<p>News body</p>", record.GetString("news"))
	assert.Equal(t, "- changed", record.GetString("key_differences"))
	assert.Equal(t, "https://example.com/spec", record.GetString("refer"))
	assert.Equal(t, "https://github.com/example/spec/compare/a...b", record.GetString("diff"))
	assert.Equal(t, "#openid4vp #eudiwallet #v2_0", record.GetString("tags"))
	assert.Equal(t, "wallet-spec-v2-released", record.GetString("canonified_title"))
}

func TestStoreNewsDraftActivityIsIdempotentByDiff(t *testing.T) {
	app := newCredimiTestApp(t)
	act := NewStoreNewsDraftActivity(app)

	first, err := executeActivity(t, act, validNewsDraft())
	require.NoError(t, err)
	firstOutput, ok := first.Output.(map[string]any)
	require.True(t, ok)
	before := countNews(t, app)

	retry := validNewsDraft()
	retry.Title = "Another title"
	second, err := executeActivity(t, act, retry)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"id": firstOutput["id"], "duplicate": true}, second.Output)
	require.Equal(t, before, countNews(t, app))

	record, err := app.FindRecordById("news", firstOutput["id"].(string))
	require.NoError(t, err)
	assert.Equal(t, "Wallet spec v2 released", record.GetString("title"))
}

func TestStoreNewsDraftActivityDuplicateOfPublishedNews(t *testing.T) {
	app := newCredimiTestApp(t)
	published := createTestNews(t, app, "Published", "", "https://example.com/diff", true)

	result, err := executeActivity(t, NewStoreNewsDraftActivity(app), StoreNewsDraftInput{
		Title: "Draft",
		Refer: "https://example.com/refer",
		Diff:  "https://example.com/diff",
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"id": published.Id, "duplicate": true}, result.Output)
}

func TestStoreNewsDraftActivityRejectsInvalidPayload(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*StoreNewsDraftInput)
	}{
		{name: "empty title", mutate: func(in *StoreNewsDraftInput) { in.Title = "  " }},
		{name: "missing diff", mutate: func(in *StoreNewsDraftInput) { in.Diff = "" }},
		{
			name:   "relative diff",
			mutate: func(in *StoreNewsDraftInput) { in.Diff = "/compare/a...b" },
		},
		{
			name:   "non http diff",
			mutate: func(in *StoreNewsDraftInput) { in.Diff = "ftp://example.com/x" },
		},
		{name: "missing refer", mutate: func(in *StoreNewsDraftInput) { in.Refer = "" }},
		{name: "invalid refer", mutate: func(in *StoreNewsDraftInput) { in.Refer = "not a url" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newCredimiTestApp(t)
			before := countNews(t, app)
			payload := validNewsDraft()
			tc.mutate(&payload)

			_, err := executeActivity(t, NewStoreNewsDraftActivity(app), payload)
			requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
			require.Equal(t, before, countNews(t, app))
		})
	}
}

func TestStoreNewsDraftActivitySaveFailureIsRetryable(t *testing.T) {
	app := newCredimiTestApp(t)
	app.OnRecordCreate("news").BindFunc(func(*core.RecordEvent) error {
		return fmt.Errorf("save failed")
	})
	_, err := executeActivity(t, NewStoreNewsDraftActivity(app), validNewsDraft())
	requireActivityError(t, err, errorcodes.DatabaseOperationFailed, false)
}

func createTestNews(
	t *testing.T,
	app *tests.TestApp,
	title, summary, diff string,
	published bool,
) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("news")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("title", title)
	record.Set("summary", summary)
	record.Set("diff", diff)
	record.Set("published", published)
	require.NoError(t, app.Save(record))
	return record
}

func searchNewsItems(t *testing.T, app core.App, payload SearchNewsInput) []map[string]any {
	t.Helper()
	result, err := executeActivity(t, NewSearchNewsActivity(app), payload)
	require.NoError(t, err)
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	rawItems, ok := output["items"].([]any)
	require.True(t, ok)
	items := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]any)
		require.True(t, ok)
		items = append(items, item)
	}
	return items
}

func newsIDs(items []map[string]any) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item["id"].(string))
	}
	return ids
}

func TestSearchNewsActivityFilters(t *testing.T) {
	app := newCredimiTestApp(t)
	draft := createTestNews(
		t,
		app,
		"OpenID4VP draft 25",
		"<p>Wallet changes</p>",
		"https://example.com/d1",
		false,
	)
	published := createTestNews(
		t,
		app,
		"EUDI release",
		"<b>OPENID4VP</b> update",
		"https://example.com/d2",
		true,
	)
	other := createTestNews(t, app, "Unrelated", "nothing", "https://example.com/d3", false)
	wildcard := createTestNews(t, app, "100% done_now", "", "https://example.com/d4", false)

	cases := []struct {
		name    string
		payload SearchNewsInput
		want    []string
	}{
		{
			name:    "no filter returns all including drafts",
			payload: SearchNewsInput{},
			want:    []string{draft.Id, published.Id, other.Id, wildcard.Id},
		},
		{
			name: "diff exact match",
			payload: SearchNewsInput{
				DiffURLs: []string{
					"https://example.com/d1",
					"https://example.com/d3",
					"https://example.com/d",
				},
			},
			want: []string{draft.Id, other.Id},
		},
		{
			name:    "query case insensitive on title and summary",
			payload: SearchNewsInput{Query: "openid4vp"},
			want:    []string{draft.Id, published.Id},
		},
		{
			name: "query or diff",
			payload: SearchNewsInput{
				Query:    "UNRELATED",
				DiffURLs: []string{"https://example.com/d1"},
			},
			want: []string{draft.Id, other.Id},
		},
		{
			name:    "query wildcards are literal",
			payload: SearchNewsInput{Query: "0% done_"},
			want:    []string{wildcard.Id},
		},
		{
			name:    "query percent alone matches literally",
			payload: SearchNewsInput{Query: "%"},
			want:    []string{wildcard.Id},
		},
		{
			name:    "no match",
			payload: SearchNewsInput{Query: "nonexistent"},
			want:    []string{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := searchNewsItems(t, app, tc.payload)
			assert.ElementsMatch(t, tc.want, newsIDs(items))
		})
	}
}

func TestSearchNewsActivityItemFields(t *testing.T) {
	app := newCredimiTestApp(t)
	longText := strings.Repeat("é", 600)
	record := createTestNews(
		t,
		app,
		"Draft",
		"<p>Hello&nbsp;<b>world</b></p>\n\n  <p>again &amp; "+longText+"</p>",
		"https://example.com/diff",
		false,
	)
	record.Set("refer", "https://example.com/refer")
	record.Set("tags", "#a #b")
	require.NoError(t, app.Save(record))

	items := searchNewsItems(
		t,
		app,
		SearchNewsInput{DiffURLs: []string{"https://example.com/diff"}},
	)
	require.Len(t, items, 1)
	item := items[0]
	summary, ok := item["summary_text"].(string)
	require.True(t, ok)
	assert.Equal(t, 500, len([]rune(summary)))
	assert.True(t, strings.HasPrefix(summary, "Hello world again & éé"), summary)
	assert.Equal(t, record.Id, item["id"])
	assert.Equal(t, "Draft", item["title"])
	assert.Equal(t, "https://example.com/refer", item["refer"])
	assert.Equal(t, "https://example.com/diff", item["diff"])
	assert.Equal(t, "#a #b", item["tags"])
	assert.Equal(t, false, item["published"])
	assert.Equal(
		t,
		record.GetDateTime("created").Time().UTC().Format(time.RFC3339),
		item["created"],
	)
}

func TestSearchNewsActivityLimit(t *testing.T) {
	app := newCredimiTestApp(t)
	for i := range 52 {
		createTestNews(
			t,
			app,
			fmt.Sprintf("News %d", i),
			"",
			fmt.Sprintf("https://example.com/%d", i),
			false,
		)
	}

	cases := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: 10},
		{name: "below minimum", limit: -3, want: 1},
		{name: "within range", limit: 7, want: 7},
		{name: "above maximum", limit: 100, want: 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := searchNewsItems(t, app, SearchNewsInput{Limit: tc.limit})
			assert.Len(t, items, tc.want)
		})
	}
}

func TestNewsSummaryText(t *testing.T) {
	assert.Equal(t, "a b c", newsSummaryText("<p>a</p>\n<p>b\t\tc</p>"))
	assert.Equal(t, strings.Repeat("x", 500), newsSummaryText(strings.Repeat("x", 501)))
	assert.Empty(t, newsSummaryText(""))
}
