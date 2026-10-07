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

// SecretsToAny converts parsed secrets to the map[string]any form carried by
// activity results and API responses. Empty secrets return nil.
func SecretsToAny(secrets map[string]string) map[string]any {
	if len(secrets) == 0 {
		return nil
	}
	out := make(map[string]any, len(secrets))
	for key, value := range secrets {
		out[key] = value
	}
	return out
}
