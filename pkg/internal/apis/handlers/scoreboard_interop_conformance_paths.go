// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"fmt"
	"strings"
)

type interopConformanceSuiteGroup struct {
	ID    string // standard/version/suite
	Title string // "standard • version • suite"
}

func interopSuiteGroupFromPath(path string) (group interopConformanceSuiteGroup, leaf string, err error) {
	segments := strings.Split(path, "/")
	if len(segments) != 4 {
		return interopConformanceSuiteGroup{}, "", errors.New("invalid conformance path")
	}
	standard, version, suite, test := segments[0], segments[1], segments[2], segments[3]
	leaf = fmt.Sprintf("%s/%s/%s/%s", standard, version, suite, test)
	return interopConformanceSuiteGroup{
		ID:    fmt.Sprintf("%s/%s/%s", standard, version, suite),
		Title: fmt.Sprintf("%s • %s • %s", standard, version, suite),
	}, leaf, nil
}
