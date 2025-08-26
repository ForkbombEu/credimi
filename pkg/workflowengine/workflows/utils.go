// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

func AsSliceOfMaps(val any) []map[string]any {
	if v, ok := val.([]map[string]any); ok {
		return v
	}
	if arr, ok := val.([]any); ok {
		res := make([]map[string]any, 0, len(arr))
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				res = append(res, m)
			}
		}
		return res
	}
	return nil
}
