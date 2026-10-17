// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Names of the activities that read and write news records for the news agent.
const (
	SearchNewsActivityName     = "Search news"
	StoreNewsDraftActivityName = "Store a news draft"
)

const (
	newsCollection          = "news"
	defaultNewsSearchLimit  = 10
	maxNewsSearchLimit      = 50
	newsSummaryTextMaxRunes = 500
)

var (
	htmlTagPattern = regexp.MustCompile(`<[^>]*>`)
	likeEscaper    = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
)

type SearchNewsInput struct {
	Query    string   `json:"query"`
	DiffURLs []string `json:"diff_urls"`
	Limit    int      `json:"limit"`
}

type NewsSearchItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	SummaryText string `json:"summary_text"`
	Refer       string `json:"refer"`
	Diff        string `json:"diff"`
	Tags        string `json:"tags"`
	Published   bool   `json:"published"`
	Created     string `json:"created"`
}

type SearchNewsOutput struct {
	Items []NewsSearchItem `json:"items"`
}

// SearchNewsActivity searches news records, unpublished drafts included.
type SearchNewsActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewSearchNewsActivity(app core.App) *SearchNewsActivity {
	return &SearchNewsActivity{
		BaseActivity: workflowengine.BaseActivity{Name: SearchNewsActivityName},
		app:          app,
	}
}

func (a *SearchNewsActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *SearchNewsActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[SearchNewsInput](input.Payload)
	if err != nil {
		return result, a.NewCodedError(errorcodes.MissingOrInvalidPayload, false, err)
	}
	items, err := searchNews(a.app, payload)
	if err != nil {
		return result, a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
	result.Output = SearchNewsOutput{Items: items}
	return result, nil
}

// searchNews returns the newest news records whose diff equals one of the
// given URLs or whose title, summary or news contain the query, ignoring case.
// With neither filter it returns the newest records.
func searchNews(app core.App, in SearchNewsInput) ([]NewsSearchItem, error) {
	var filters []dbx.Expression
	diffURLs := make([]any, 0, len(in.DiffURLs))
	for _, diffURL := range in.DiffURLs {
		if diffURL != "" {
			diffURLs = append(diffURLs, diffURL)
		}
	}
	if len(diffURLs) > 0 {
		filters = append(filters, dbx.In("diff", diffURLs...))
	}
	if query := strings.TrimSpace(in.Query); query != "" {
		filters = append(filters, dbx.NewExp(
			`LOWER([[title]]) LIKE {:query} ESCAPE '\' OR `+
				`LOWER([[summary]]) LIKE {:query} ESCAPE '\' OR `+
				`LOWER([[news]]) LIKE {:query} ESCAPE '\'`,
			dbx.Params{"query": "%" + likeEscaper.Replace(strings.ToLower(query)) + "%"},
		))
	}

	q := app.RecordQuery(newsCollection).
		OrderBy("created DESC", "id DESC").
		Limit(int64(clampNewsSearchLimit(in.Limit)))
	if len(filters) > 0 {
		q = q.AndWhere(dbx.Or(filters...))
	}
	var records []*core.Record
	if err := q.All(&records); err != nil {
		return nil, fmt.Errorf("search news: %w", err)
	}

	items := make([]NewsSearchItem, 0, len(records))
	for _, record := range records {
		items = append(items, NewsSearchItem{
			ID:          record.Id,
			Title:       record.GetString("title"),
			SummaryText: newsSummaryText(record.GetString("summary")),
			Refer:       record.GetString("refer"),
			Diff:        record.GetString("diff"),
			Tags:        record.GetString("tags"),
			Published:   record.GetBool("published"),
			Created:     record.GetDateTime("created").Time().UTC().Format(time.RFC3339),
		})
	}
	return items, nil
}

func clampNewsSearchLimit(limit int) int {
	switch {
	case limit == 0:
		return defaultNewsSearchLimit
	case limit < 1:
		return 1
	case limit > maxNewsSearchLimit:
		return maxNewsSearchLimit
	default:
		return limit
	}
}

// newsSummaryText strips HTML tags from summary, collapses whitespace and
// truncates the result to newsSummaryTextMaxRunes runes.
func newsSummaryText(summary string) string {
	text := html.UnescapeString(htmlTagPattern.ReplaceAllString(summary, " "))
	text = strings.Join(strings.Fields(text), " ")
	if runes := []rune(text); len(runes) > newsSummaryTextMaxRunes {
		text = string(runes[:newsSummaryTextMaxRunes])
	}
	return text
}

type StoreNewsDraftInput struct {
	Title            string   `json:"title"`
	SummaryHTML      string   `json:"summary_html"`
	NewsHTML         string   `json:"news_html"`
	KeyDifferencesMD string   `json:"key_differences_md"`
	Tags             []string `json:"tags"`
	Refer            string   `json:"refer"`
	Diff             string   `json:"diff"`
}

type StoreNewsDraftOutput struct {
	ID        string `json:"id"`
	Duplicate bool   `json:"duplicate"`
}

// StoreNewsDraftActivity stores an unpublished news draft, once per diff URL.
type StoreNewsDraftActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewStoreNewsDraftActivity(app core.App) *StoreNewsDraftActivity {
	return &StoreNewsDraftActivity{
		BaseActivity: workflowengine.BaseActivity{Name: StoreNewsDraftActivityName},
		app:          app,
	}
}

func (a *StoreNewsDraftActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StoreNewsDraftActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[StoreNewsDraftInput](input.Payload)
	if err == nil {
		err = validateNewsDraft(payload)
	}
	if err != nil {
		return result, a.NewCodedError(errorcodes.MissingOrInvalidPayload, false, err)
	}
	output, err := storeNewsDraft(a.app, payload)
	if err != nil {
		return result, a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
	result.Output = output
	return result, nil
}

func validateNewsDraft(in StoreNewsDraftInput) error {
	if strings.TrimSpace(in.Title) == "" {
		return errors.New("title is required")
	}
	if err := validateHTTPURL(in.Diff); err != nil {
		return fmt.Errorf("diff: %w", err)
	}
	if err := validateHTTPURL(in.Refer); err != nil {
		return fmt.Errorf("refer: %w", err)
	}
	return nil
}

func validateHTTPURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url %q: %w", raw, err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("%q is not an absolute http(s) url", raw)
	}
	return nil
}

// storeNewsDraft creates an unpublished news record unless one with the same
// diff already exists, in which case it returns that record as a duplicate.
func storeNewsDraft(app core.App, in StoreNewsDraftInput) (StoreNewsDraftOutput, error) {
	existing, err := app.FindFirstRecordByFilter(
		newsCollection,
		"diff = {:diff}",
		dbx.Params{"diff": in.Diff},
	)
	switch {
	case err == nil:
		return StoreNewsDraftOutput{ID: existing.Id, Duplicate: true}, nil
	case !errors.Is(err, sql.ErrNoRows):
		return StoreNewsDraftOutput{}, fmt.Errorf("find news by diff: %w", err)
	}

	collection, err := app.FindCollectionByNameOrId(newsCollection)
	if err != nil {
		return StoreNewsDraftOutput{}, fmt.Errorf("find news collection: %w", err)
	}
	record := core.NewRecord(collection)
	record.Set("title", strings.TrimSpace(in.Title))
	record.Set("summary", in.SummaryHTML)
	record.Set("news", in.NewsHTML)
	record.Set("key_differences", in.KeyDifferencesMD)
	record.Set("refer", in.Refer)
	record.Set("diff", in.Diff)
	record.Set("tags", normalizeNewsTags(in.Tags))
	record.Set("published", false)
	if err := app.Save(record); err != nil {
		return StoreNewsDraftOutput{}, fmt.Errorf("save news draft: %w", err)
	}
	return StoreNewsDraftOutput{ID: record.Id}, nil
}

// normalizeNewsTags lowercases tags, keeps only [a-z0-9_], drops empty and
// repeated tags and joins them as "#a #b".
func normalizeNewsTags(tags []string) string {
	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		var b strings.Builder
		for _, r := range strings.ToLower(tag) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
				b.WriteRune(r)
			}
		}
		clean := b.String()
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		normalized = append(normalized, "#"+clean)
	}
	return strings.Join(normalized, " ")
}
