// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ParseSecretsYAML parses a YAML mapping of secret names to values. An empty
// string returns nil, nil.
func ParseSecretsYAML(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	var secrets map[string]string
	if err := yaml.Unmarshal([]byte(s), &secrets); err != nil {
		return nil, fmt.Errorf("parse secrets yaml: %w", err)
	}
	return secrets, nil
}
