// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package canonify

import (
	"database/sql"
	"errors"
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
	PathLength      int
}

var CanonifyPaths = map[string]PathTemplate{
	"users": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		PathLength:      1,
	},
	"organizations": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		PathLength:      1,
	},
	"credential_issuers": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
		PathLength:      2,
	},
	"credentials": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "credential_issuers", Field: "credential_issuer"},
		PathLength:      3,
	},
	"custom_checks": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
		PathLength:      2,
	},
	"wallets": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
		PathLength:      2,
	},
	"wallet_actions": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "wallets", Field: "wallet"},
		PathLength:      3,
	},
	"wallet_versions": {
		Field:           "tag",
		CanonifiedField: "canonified_tag",
		Parent:          &Parent{Collection: "wallets", Field: "wallet"},
		PathLength:      3,
	},
	"verifiers": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
		PathLength:      2,
	},
	"use_cases_verifications": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "verifiers", Field: "verifier"},
		PathLength:      3,
	},
	"pipelines": {
		Field:           "name",
		CanonifiedField: "canonified_name",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
		PathLength:      2,
	},
	"pipeline_results": {
		Field:           "workflow_id",
		CanonifiedField: "canonified_identifier",
		Parent:          &Parent{Collection: "organizations", Field: "owner"},
		PathLength:      2,
	},
	"news": {
		Field:           "title",
		CanonifiedField: "canonified_title",
		PathLength:      1,
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

func Resolve(app core.App, path string) (*core.Record, error) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	// Only try collections with matching path length
	for collection, tpl := range CanonifyPaths {
		if tpl.PathLength != len(segments) {
			continue
		}

		chain, err := getPathChain(collection)
		if err != nil {
			continue
		}

		var rec *core.Record
		var parentID string
		match := true

		for i, col := range chain {
			tpl := CanonifyPaths[col]

			filter := fmt.Sprintf("%s = {:value}", tpl.CanonifiedField)
			params := map[string]any{"value": segments[i]}

			if i > 0 {
				filter += fmt.Sprintf(" && %s = {:parentID}", tpl.Parent.Field)
				params["parentID"] = parentID
			}

			r, err := app.FindFirstRecordByFilter(col, filter, params)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					return nil, err
				}
				match = false
				break
			}

			rec = r
			parentID = rec.Id
		}

		if match {
			return rec, nil
		}
	}

	return nil, sql.ErrNoRows
}

func Validate(app core.App, path string) (*core.Record, error) {
	if path == "" || path == "/" {
		return nil, fmt.Errorf("empty path")
	}
	rec, err := Resolve(app, path)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", path, err)
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
