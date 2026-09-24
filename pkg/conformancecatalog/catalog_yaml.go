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

func readYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		// Check files may not be YAML metadata-first; ignore parse errors for title extraction.
		if _, ok := out.(*checkFileMeta); ok {
			return nil
		}
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return nil
}
