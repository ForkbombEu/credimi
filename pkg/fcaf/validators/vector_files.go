// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type validatorVectors struct {
	Positive []string `json:"positive" yaml:"positive"`
	Negative []string `json:"negative" yaml:"negative"`
}

type vectorFile struct {
	Cases []vectorCase `json:"cases" yaml:"cases"`
}

type vectorCase struct {
	ID          string `json:"id"           yaml:"id"`
	Text        string `json:"text"         yaml:"text"`
	BytesBase64 string `json:"bytes_base64" yaml:"bytes_base64"`
}

func decodeVectors(params map[string]any) (validatorVectors, error) {
	decoded, err := DecodeParams[struct {
		Vectors validatorVectors `json:"vectors"`
	}](params)
	if err != nil {
		return validatorVectors{}, err
	}
	return decoded.Vectors, nil
}

func loadVectorFile(path string) (vectorFile, error) {
	resolved, err := resolveVectorPath(path)
	if err != nil {
		return vectorFile{}, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return vectorFile{}, fmt.Errorf("read vector file %q: %w", resolved, err)
	}
	var file vectorFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return vectorFile{}, fmt.Errorf("parse vector file %q: %w", resolved, err)
	}
	return file, nil
}

func vectorCaseBytes(tc vectorCase) ([]byte, error) {
	switch {
	case tc.BytesBase64 != "":
		decoded, err := base64.StdEncoding.DecodeString(tc.BytesBase64)
		if err != nil {
			return nil, fmt.Errorf("decode bytes_base64 for case %q: %w", tc.ID, err)
		}
		return decoded, nil
	case tc.Text != "":
		return []byte(tc.Text), nil
	default:
		return nil, fmt.Errorf("vector case %q must define text or bytes_base64", tc.ID)
	}
}

func resolveVectorPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve vector path %q: %w", path, err)
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("resolve vector path %q: file not found", path)
}
