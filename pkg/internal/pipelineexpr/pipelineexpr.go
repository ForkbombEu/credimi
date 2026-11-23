// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package pipelineexpr holds the pipeline expression syntax and the default
// FCAF fixture values, with no dependency beyond the standard library, so
// lightweight packages (pkg/fcaf/walletroles) can read pipelines without
// pulling in the pipeline engine and Temporal.
package pipelineexpr

import (
	"regexp"
	"strings"
)

// Pattern matches a ${{ ... }} expression; its first group is the body.
var Pattern = regexp.MustCompile(`\${{\s*([a-zA-Z0-9_\-\.\[\]\|\s\(\):,]+?)\s*}}`)

// ExpressionRefs returns the trimmed bodies of every ${{ ... }} expression in s.
func ExpressionRefs(s string) []string {
	matches := Pattern.FindAllStringSubmatch(s, -1)
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		refs = append(refs, strings.TrimSpace(match[1]))
	}
	return refs
}

// The ${fixture.<key>} values a pipeline gets when its runtime.fixture does
// not set them.
const (
	DefaultIssuerURL   = "https://issuer-backend.eudiw.dev"
	DefaultVerifierURL = "https://capture-wallet.credimi.io"
	DefaultLogChecker  = "eudiw"
)

// FixtureDefaults returns the default fixture values keyed by fixture key.
func FixtureDefaults() map[string]string {
	return map[string]string{
		"issuer_url":   DefaultIssuerURL,
		"verifier_url": DefaultVerifierURL,
		"log_checker":  DefaultLogChecker,
	}
}
