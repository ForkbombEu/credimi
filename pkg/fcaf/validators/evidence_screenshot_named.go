// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/tools/inflector"
)

// screenshotStemLimit mirrors the cap Credimi's screenshot storage applies to a
// file name. A stored name is
// `Snakecase(<pipeline step id>-<Maestro screenshot name>)_<random>.<ext>`, and
// PocketBase keeps only the first 100 characters of the snake-cased stem before
// appending the random part (tools/filesystem/file.go). A screenshot name that
// does not fit in the room its step id leaves behind is cut off in the stored
// name and cannot be proven by name, so a name this validator is asked for must
// satisfy `len(Snakecase(stepID)) + 1 + len(Snakecase(name)) <= 100`. When a
// stored stem reaches the cap and does not carry the wanted name, the name may
// have been truncated away, so the validator reports blocked rather than fail.
const screenshotStemLimit = 100

// EvidenceScreenshotNamedValidator proves that a flow stored a screenshot under
// a given name, which is how a source requirement that a particular screen was
// shown is evidenced: the flow takes that screenshot only while the screen is
// up, and the stored file name carries the name.
//
// The evidence is the mobile step's `maestro_screenshot_urls`, so the check is
// on the file names of those URLs. The name is normalised with the same
// Snakecase PocketBase applies when it stores the file, so `no-match-notice`
// matches `..._no_match_notice_<random>.png`.
type EvidenceScreenshotNamedValidator struct{}

// ID returns the validator identifier.
func (EvidenceScreenshotNamedValidator) ID() string {
	return "evidence.screenshot_named"
}

type evidenceScreenshotNamedParams struct {
	Name string `json:"name"`
}

// Validate requires at least one stored screenshot whose file name carries the
// given name.
func (EvidenceScreenshotNamedValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[evidenceScreenshotNamedParams](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if strings.TrimSpace(params.Name) == "" {
		return Result{Status: StatusError, Message: "name param is required"}
	}

	names, result := screenshotFileNames(input.Value)
	if result != nil {
		return *result
	}

	wanted := inflector.Snakecase(params.Name)
	truncated := false
	for _, name := range names {
		if strings.Contains(name, wanted) {
			return Result{
				Status: StatusPass,
				Message: fmt.Sprintf(
					"a screenshot named %q was stored (%s)",
					params.Name,
					name,
				),
			}
		}
		if stem, ok := screenshotStoredStem(name); ok && len(stem) >= screenshotStemLimit {
			truncated = true
		}
	}

	// A stored name whose stem is exactly the cap may have lost the name to
	// truncation, so the screen cannot be disproven by name. That is a storage
	// limitation, not a Wallet breach, and reports blocked.
	if truncated {
		return Result{
			Status: StatusBlocked,
			Message: fmt.Sprintf(
				"cannot tell whether a screenshot named %q was stored: the stem of %s "+
					"is cut off at PocketBase's %d-character cap, which is where the "+
					"name would have been",
				params.Name,
				strings.Join(names, ", "),
				screenshotStemLimit,
			),
		}
	}

	return Result{
		Status: StatusFail,
		Message: fmt.Sprintf(
			"no screenshot named %q was stored; the %d screenshot(s) are %s "+
				"(a stored name keeps only the first %d characters of "+
				"Snakecase(step id + screenshot name), so a name whose step id "+
				"leaves too little room is truncated away)",
			params.Name,
			len(names),
			strings.Join(names, ", "),
			screenshotStemLimit,
		),
	}
}

// screenshotStoredStem returns the stem PocketBase kept of a stored screenshot
// file name. A stored name is `<stem>_<10 random characters><extension>`, so the
// extension and the random part are stripped; a stem of screenshotStemLimit
// characters is one that reached PocketBase's cap and may have been truncated.
func screenshotStoredStem(fileName string) (string, bool) {
	stem := fileName
	if index := strings.LastIndexByte(stem, '.'); index >= 0 {
		stem = stem[:index]
	}
	index := strings.LastIndexByte(stem, '_')
	if index < 0 {
		return "", false
	}
	return stem[:index], true
}

// screenshotFileNames returns the file names of the screenshot URLs in the
// evidence. A single URL is accepted as a one-screenshot list; a missing
// evidence or a non-URL value is a failure to prove the screen, matching the
// way evidence.non_empty reports missing evidence.
func screenshotFileNames(value any) ([]string, *Result) {
	var urls []string
	switch typed := value.(type) {
	case []any:
		urls = make([]string, 0, len(typed))
		for index, item := range typed {
			url, ok := item.(string)
			if !ok {
				return nil, &Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"evidence[%d] is %T, expected a screenshot URL",
						index,
						item,
					),
				}
			}
			urls = append(urls, url)
		}
	case []string:
		urls = typed
	case string:
		urls = []string{typed}
	default:
		return nil, &Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"evidence value is %T, expected a list of screenshot URLs",
				value,
			),
		}
	}
	if len(urls) == 0 {
		return nil, &Result{
			Status:  StatusFail,
			Message: "evidence contains no screenshot",
		}
	}

	names := make([]string, 0, len(urls))
	for _, url := range urls {
		names = append(names, screenshotURLFileName(url))
	}
	return names, nil
}

// screenshotURLFileName returns the stored file name of a screenshot URL,
// ignoring a query string or fragment the way the workflow's own URL handling
// does.
func screenshotURLFileName(rawURL string) string {
	name := rawURL
	if index := strings.IndexAny(name, "?#"); index >= 0 {
		name = name[:index]
	}
	if index := strings.LastIndexByte(name, '/'); index >= 0 {
		name = name[index+1:]
	}
	return name
}
