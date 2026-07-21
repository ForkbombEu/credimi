// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package catalog

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
)

type Catalog struct {
	Tests         map[string]dsl.TestDefinition
	Preconditions map[string]dsl.PreconditionDefinition
}

func Load(root string) (*Catalog, error) {
	tests, err := LoadTests(filepath.Join(root, "tests"))
	if err != nil {
		return nil, err
	}
	preconditions, err := LoadPreconditions(filepath.Join(root, "preconditions"))
	if err != nil {
		return nil, err
	}
	return &Catalog{
		Tests:         tests,
		Preconditions: preconditions,
	}, nil
}

func LoadTests(root string) (map[string]dsl.TestDefinition, error) {
	tests := map[string]dsl.TestDefinition{}
	err := walkYAML(root, func(path string) error {
		if strings.Contains(
			path,
			string(filepath.Separator)+"_implementation"+string(filepath.Separator),
		) {
			return nil
		}
		def, err := dsl.ParseFile(path)
		if err != nil {
			return err
		}
		if _, exists := tests[def.ID]; exists {
			return fmt.Errorf("duplicate fcaf test id %q in %s", def.ID, path)
		}
		tests[def.ID] = *def
		return nil
	})
	return tests, err
}

func LoadTestsByID(root string, ids []string) (map[string]dsl.TestDefinition, error) {
	tests, err := LoadTests(root)
	if err != nil {
		return nil, err
	}
	selected := map[string]dsl.TestDefinition{}
	for _, id := range ids {
		def, ok := tests[id]
		if !ok {
			return nil, fmt.Errorf("fcaf test id %q not found", id)
		}
		selected[id] = def
	}
	return selected, nil
}

func (c *Catalog) ResolveSelectedTests(
	testIDs []string,
	suite string,
	runtime map[string]any,
) ([]string, error) {
	if c == nil {
		return nil, fmt.Errorf("catalog is nil")
	}

	selected := map[string]struct{}{}
	if len(testIDs) == 0 {
		for id, test := range c.Tests {
			if suite != "" && !matchesSuite(test, suite) {
				continue
			}
			if !matchesApplicability(test, runtime) {
				continue
			}
			selected[id] = struct{}{}
		}
	} else {
		for _, id := range testIDs {
			if _, ok := c.Tests[id]; !ok {
				return nil, fmt.Errorf("fcaf test id %q not found", id)
			}
			selected[id] = struct{}{}
		}
	}

	expanded := map[string]struct{}{}
	visiting := map[string]struct{}{}
	for id := range selected {
		if err := c.expandTest(id, expanded, visiting); err != nil {
			return nil, err
		}
	}

	out := make([]string, 0, len(expanded))
	for id := range expanded {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func LoadPreconditions(root string) (map[string]dsl.PreconditionDefinition, error) {
	preconditions := map[string]dsl.PreconditionDefinition{}
	err := walkYAML(root, func(path string) error {
		def, err := dsl.ParsePreconditionFile(path)
		if err != nil {
			return err
		}
		if _, exists := preconditions[def.ID]; exists {
			return fmt.Errorf("duplicate fcaf precondition id %q in %s", def.ID, path)
		}
		preconditions[def.ID] = *def
		return nil
	})
	return preconditions, err
}

func walkYAML(root string, visit func(path string) error) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !slices.Contains([]string{".yaml", ".yml"}, ext) {
			return nil
		}
		return visit(path)
	})
}

func (c *Catalog) expandTest(
	id string,
	expanded map[string]struct{},
	visiting map[string]struct{},
) error {
	if _, ok := expanded[id]; ok {
		return nil
	}
	if _, ok := visiting[id]; ok {
		return fmt.Errorf("cycle detected while expanding test %q", id)
	}
	test, ok := c.Tests[id]
	if !ok {
		return fmt.Errorf("fcaf test id %q not found", id)
	}
	visiting[id] = struct{}{}
	for _, ref := range test.Preconditions {
		if strings.HasPrefix(ref.Ref, "test.") {
			if err := c.expandTest(
				strings.TrimPrefix(ref.Ref, "test."),
				expanded,
				visiting,
			); err != nil {
				return err
			}
			continue
		}
		if _, ok := c.Preconditions[ref.Ref]; !ok {
			return fmt.Errorf("fcaf precondition %q not found", ref.Ref)
		}
	}
	delete(visiting, id)
	expanded[id] = struct{}{}
	return nil
}

func matchesSuite(test dsl.TestDefinition, suite string) bool {
	current := test.Suite.SUT + "/" + test.Suite.Role
	return current == suite || strings.HasPrefix(current+"/"+test.Suite.Section, suite)
}

func matchesApplicability(test dsl.TestDefinition, runtime map[string]any) bool {
	for key, value := range test.Applicability {
		runtimeValue, ok := runtime[key]
		if !ok {
			continue
		}
		if !reflect.DeepEqual(runtimeValue, value) {
			return false
		}
	}
	return true
}
