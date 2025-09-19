// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package canonify

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// CanonifyOptions controls behaviour of Canonify.
type CanonifyOptions struct {
	Separator   rune
	MinLen      int
	Fallback    string
	MaxAttempts int
}

var DefaultOptions = CanonifyOptions{
	Separator:   '_',
	MinLen:      1,
	Fallback:    "item_name",
	MaxAttempts: 1000000,
}

// ExistsFunc is the callback used to ask if a candidate already exists in DB.
// Should be deterministic.
type ExistsFunc func(name string) bool

// ErrExhaustedAttempts returned when we couldn't find a free name within MaxAttempts.
var ErrExhaustedAttempts = errors.New("could not find unique name: exhausted attempts")

// Canonify converts an arbitrary input string into a deterministic, compact, lowercased,
// ASCII-friendly slug and ensures uniqueness by consulting exists(name).
//
// The exists callback will be called for each candidate (base name and sequential suffixes).
func Canonify(in string, exists ExistsFunc) (string, error) {
	return CanonifyWithOptions(in, exists, DefaultOptions)
}

// CanonifyWithOptions same as Canonify but with configurable options.
func CanonifyWithOptions(in string, exists ExistsFunc, opts CanonifyOptions) (string, error) {
	if exists == nil {
		return "", errors.New("exists func is required")
	}
	sep := opts.Separator
	if sep == 0 {
		sep = DefaultOptions.Separator
	}
	// Step 1: Remove invisible/control runes + normalize (NFKD).
	clean := removeInvisibleAndControl(in)

	// Step 2: Normalize and remove diacritics (NFKD), then keep base ASCII letters/digits, map others to sep.
	clean = norm.NFKD.String(clean)

	// Build rune-by-rune, mapping
	var b strings.Builder
	prevSep := false
	for _, r := range strings.ToLower(clean) {
		// Skip combining marks (diacritics) and non-graphic
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSep = false
			continue
		}
		// For ASCII separator characters allowed (we map them to sep)
		// For everything else (punctuation, spaces, symbols, non-ASCII letters after stripping), map to sep
		if !prevSep {
			b.WriteRune(sep)
			prevSep = true
		}
	}
	out := b.String()

	// collapse multiple separators (already done via prevSep logic), trim leading/trailing
	out = strings.Trim(out, string(sep))

	// if empty, use fallback
	if len(out) < opts.MinLen {
		if opts.Fallback == "" {
			opts.Fallback = DefaultOptions.Fallback
		}
		out = opts.Fallback
	}

	// Now check uniqueness deterministically: try out, out-1, out-2, ...
	if !exists(out) {
		return out, nil
	}

	// Try suffixes deterministically
	// We'll append "-<counter>" using decimal; this is simple, deterministic and readable.
	for i := 1; i <= opts.MaxAttempts; i++ {
		candidate := fmt.Sprintf("%s%c%d", out, sep, i)
		if !exists(candidate) {
			return candidate, nil
		}
	}
	return "", ErrExhaustedAttempts
}

// removeInvisibleAndControl removes invisible, zero-width, and control characters
// while preserving visible text and common whitespace. We normalize various
// Unicode invisible characters to nothing. This is a conservative list but covers
// the usual suspects.
func removeInvisibleAndControl(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' || r == ' ' {
			b.WriteRune(' ')
			continue
		}
		switch r {
		case '\u200B', // ZERO WIDTH SPACE
			'\u200C',                     // ZERO WIDTH NON-JOINER
			'\u200D',                     // ZERO WIDTH JOINER
			'\uFEFF',                     // ZERO WIDTH NO-BREAK SPACE / BOM
			'\u2060',                     // WORD JOINER
			'\u2061', '\u2062', '\u2063', // function application, invisible times, invisible separator
			'\u180E': // MONGOLIAN VOWEL SEPARATOR (deprecated but appears)
			continue
		}
		if unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
