// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"net/url"
	"path"
	"sort"
	"strings"
)

// ImageReferenceURLs collects every image reference URL inside an evidence
// value, depth-first, deduplicated and in stable order.
func ImageReferenceURLs(value any) []string {
	seen := map[string]struct{}{}
	collectImageReferences(value, seen)
	urls := make([]string, 0, len(seen))
	for reference := range seen {
		urls = append(urls, reference)
	}
	sort.Strings(urls)
	return urls
}

func collectImageReferences(value any, seen map[string]struct{}) {
	switch typed := value.(type) {
	case string:
		if isImageReference(typed) {
			seen[typed] = struct{}{}
		}
	case []any:
		for _, child := range typed {
			collectImageReferences(child, seen)
		}
	case []string:
		for _, child := range typed {
			collectImageReferences(child, seen)
		}
	case map[string]any:
		for _, child := range typed {
			collectImageReferences(child, seen)
		}
	}
}

func isImageReference(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".gif", ".jpeg", ".jpg", ".png", ".webp":
		return true
	default:
		return false
	}
}
