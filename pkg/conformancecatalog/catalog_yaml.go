// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// walk-only YAML shapes (not exposed as nested API DTOs).
type standardYAML struct {
	UID string `yaml:"uid"`
}

type versionYAML struct {
	UID string `yaml:"uid"`
}

type suiteYAML struct {
	UID         string   `yaml:"uid"`
	Name        string   `yaml:"name"`
	Homepage    string   `yaml:"homepage"`
	Repository  string   `yaml:"repository"`
	Help        string   `yaml:"help"`
	Description string   `yaml:"description"`
	Logo        string   `yaml:"logo"`
	VisibleIn   []string `yaml:"visible_in"`
	Protocol    string   `yaml:"protocol"`
	SUT         string   `yaml:"sut"`
	Role        string   `yaml:"role"`
	Provider    string   `yaml:"provider"`
}

// suiteDisplayFields are authored suite metadata keyed for suite projection
// (ADR-0002), not denormalized onto checks.
type suiteDisplayFields struct {
	Name        string
	Homepage    string
	Repository  string
	Help        string
	Description string
	Logo        string
}

func suiteDisplayFromYAML(s suiteYAML) suiteDisplayFields {
	return suiteDisplayFields{
		Name:        strings.TrimSpace(s.Name),
		Homepage:    strings.TrimSpace(s.Homepage),
		Repository:  strings.TrimSpace(s.Repository),
		Help:        strings.TrimSpace(s.Help),
		Description: strings.TrimSpace(s.Description),
		Logo:        strings.TrimSpace(s.Logo),
	}
}

type checkFileMeta struct {
	Title    string `yaml:"title"`
	Name     string `yaml:"name"`
	Protocol string `yaml:"protocol"`
	SUT      string `yaml:"sut"`
	Role     string `yaml:"role"`
	Provider string `yaml:"provider"`
}

func titleFromMeta(meta checkFileMeta, stem string) string {
	if title := strings.TrimSpace(meta.Title); title != "" {
		return title
	}
	if name := strings.TrimSpace(meta.Name); name != "" {
		return name
	}
	return filenameTitle(stem)
}

func filenameTitle(stem string) string {
	if stem == "" {
		return "untitled"
	}
	return stem
}

// readRequiredYAML loads authored YAML (standard/version/suite/FCAF defs).
// Missing file is OK (callers Stat-gate when the file is mandatory). Unmarshal
// errors fail closed.
func readRequiredYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return nil
}

// readBestEffortCheckMeta extracts title/facets from a classic check file.
// Missing or non-metadata-first YAML yields zero values (fail open).
func readBestEffortCheckMeta(path string) checkFileMeta {
	data, err := os.ReadFile(path)
	if err != nil {
		return checkFileMeta{}
	}
	var meta checkFileMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return checkFileMeta{}
	}
	return meta
}
