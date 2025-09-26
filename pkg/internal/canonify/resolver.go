// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package canonify

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

type Parent struct {
	Collection string
	Field      string
}
type PathTemplate struct {
	Field           string
	CanonifiedField string
	Parent          *Parent
}

var CanonifyPaths = map[string]PathTemplate{
	"users": {
		Field:           "name",
		CanonifiedField: "canonified_name",
	},
	"organizations": {
		Field:           "name",
		CanonifiedField: "canonified_name",
	},
	"credential_issuers": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
	},
	"credentials": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "credential_issuers", Field: "credential_issuer"},
	},
	"custom_checks": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
	},
	"wallets": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
	},
	"wallet_actions": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "wallets", Field: "wallet"},
	},
	"verifiers": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
	},
	"use_cases_verifications": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "verifiers", Field: "verifier"},
	},
	"news": {
		Field:           "title",
		CanonifiedField: "canonified_title",
	},
}

// BuildPath constructs the path for a record
func BuildPath(
	app core.App,
	rec *core.Record,
	tpl PathTemplate,
	candidateName string,
) (string, error) {
	parts := []string{}

	if tpl.Parent != nil {
		parentID := rec.GetString(tpl.Parent.Field)
		if parentID == "" {
			return "", fmt.Errorf(
				"missing parent %s on %s",
				tpl.Parent.Field,
				rec.Collection().Name,
			)
		}
		parentTpl, ok := CanonifyPaths[tpl.Parent.Collection]
		if !ok {
			return "", fmt.Errorf(
				"no path template for parent collection %s",
				tpl.Parent.Collection,
			)
		}
		parentRec, err := app.FindRecordById(tpl.Parent.Collection, parentID)
		if err != nil {
			return "", fmt.Errorf("failed to load parent %s: %w", tpl.Parent.Collection, err)
		}
		parentPath, err := BuildPath(app, parentRec, parentTpl, "")
		if err != nil {
			return "", err
		}
		parts = append(parts, strings.Trim(parentPath, "/"))
	}
	val := rec.GetString(tpl.CanonifiedField)
	if candidateName != "" {
		val = candidateName
	}
	parts = append(parts, val)

	return "/" + strings.Join(parts, "/"), nil
}

func Resolve(app core.App, collection, path string) (*core.Record, error) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 {
		return nil, fmt.Errorf("empty path")
	}
	chain, err := getPathChain(collection)
	if err != nil {
		return nil, err
	}
	var parentID string
	var rec *core.Record

	for i, col := range chain {
		tpl, ok := CanonifyPaths[col]
		if !ok {
			return nil, fmt.Errorf("no path template for collection %s", collection)
		}
		filter := fmt.Sprintf(`%s = {:value}`, tpl.CanonifiedField)
		params := map[string]any{"value": segments[i]}

		if i > 0 {
			filter += fmt.Sprintf(` && %s = {:parentID}`, tpl.Parent.Field)
			params["parentID"] = parentID
		}

		r, err := app.FindFirstRecordByFilter(col, filter, params)
		if err != nil {
			return nil, err
		}

		rec = r
		parentID = rec.Id
	}

	return rec, nil
}

func Validate(app core.App, collection, path string) (*core.Record, error) {
	if path == "" || path == "/" {
		return nil, fmt.Errorf("empty path")
	}
	rec, err := Resolve(app, collection, path)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q for collection %q: %w", path, collection, err)
	}

	return rec, nil
}

func getPathChain(collection string) ([]string, error) {
	var chain []string
	cur := collection
	for {
		tpl, ok := CanonifyPaths[cur]
		if !ok {
			return nil, fmt.Errorf("no path template for collection %s", cur)
		}
		chain = append(chain, cur)
		if tpl.Parent == nil {
			break
		}
		cur = tpl.Parent.Collection
	}
	slices.Reverse(chain)
	return chain, nil
}
