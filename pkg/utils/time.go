// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import "time"

// ParseTimeString parses timestamps produced by Temporal and PocketBase APIs.
func ParseTimeString(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

// CompareTimeStrings compares timestamp strings chronologically.
func CompareTimeStrings(left, right string) int {
	leftTime, leftErr := ParseTimeString(left)
	rightTime, rightErr := ParseTimeString(right)
	switch {
	case leftErr == nil && rightErr == nil:
		switch {
		case leftTime.Before(rightTime):
			return -1
		case leftTime.After(rightTime):
			return 1
		default:
			return 0
		}
	case leftErr == nil:
		return 1
	case rightErr == nil:
		return -1
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

// TimeStringBefore reports whether left is chronologically before right.
func TimeStringBefore(left, right string) bool {
	return CompareTimeStrings(left, right) < 0
}

// TimeStringAfter reports whether left is chronologically after right.
func TimeStringAfter(left, right string) bool {
	return CompareTimeStrings(left, right) > 0
}
