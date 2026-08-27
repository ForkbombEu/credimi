// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package catalog

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/stretchr/testify/require"
)

var prohibitedCategoryPreconditions = map[string]struct{}{
	"pipeline.dcql.protocol-messages":      {},
	"pipeline.dcql.metadata":               {},
	"pipeline.dcql.main-interaction":       {},
	"pipeline.dcql.rp-integrity":           {},
	"pipeline.dcql.standard-all":           {},
	"pipeline.dcql.encoding":               {},
	"pipeline.dcql.credential-formats":     {},
	"pipeline.dcql.trust-mechanisms":       {},
	"pipeline.dcql.session-encryption":     {},
	"pipeline.dcql.interaction-metadata":   {},
	"pipeline.dcql.cryptography":           {},
	"pipeline.dcql.device-binding":         {},
	"pipeline.dcql.interaction-supportive": {},
}

func TestRunnableCatalogRejectsCategoryPreconditions(t *testing.T) {
	cat, err := Load(generatedWalletRelyingPartyRoot())
	require.NoError(t, err)

	for id, test := range cat.Tests {
		for _, ref := range test.Preconditions {
			_, prohibited := prohibitedCategoryPreconditions[ref.Ref]
			require.Falsef(
				t,
				prohibited,
				"runnable test %s references category precondition %s",
				id,
				ref.Ref,
			)
		}
	}
}

func TestRunnableCatalogPipelineArtifactsResolveByIdentifier(t *testing.T) {
	root := generatedWalletRelyingPartyRoot()
	cat, err := Load(root)
	require.NoError(t, err)

	for testID, test := range cat.Tests {
		for _, ref := range test.Preconditions {
			precondition, exists := cat.Preconditions[ref.Ref]
			require.Truef(t, exists, "%s references unknown precondition %s", testID, ref.Ref)
			if precondition.Kind != "pipeline" {
				continue
			}

			pipelineName := filepath.Base(
				strings.TrimSuffix(precondition.PipelineID, "/"),
			) + ".yaml"
			require.FileExistsf(
				t,
				filepath.Join(root, "pipelines", pipelineName),
				"%s pipeline precondition %s does not resolve by pipeline_id",
				testID,
				ref.Ref,
			)
		}
	}
}

func TestRemediationClassification(t *testing.T) {
	root := generatedWalletRelyingPartyRoot()
	cat, err := Load(root)
	require.NoError(t, err)

	pending := loadDefinitionsFromDirectory(
		t,
		filepath.Join(root, "tests", "_implementation", "pending"),
	)
	verifierBlocked := loadDefinitionsFromDirectory(
		t,
		filepath.Join(root, "tests", "_implementation", "verifier-blocked"),
	)

	require.Len(t, cat.Tests, 223)
	require.Len(t, pending, 280)
	require.Len(t, verifierBlocked, 56)
	require.Equal(t, 559, len(cat.Tests)+len(pending)+len(verifierBlocked))

	all := make(map[string]dsl.TestDefinition, 559)
	for _, definitions := range []map[string]dsl.TestDefinition{
		cat.Tests,
		pending,
		verifierBlocked,
	} {
		for id, definition := range definitions {
			_, duplicate := all[id]
			require.Falsef(t, duplicate, "duplicate FCAF definition %s", id)
			all[id] = definition
		}
	}

	sourceRoot := filepath.Join(
		root,
		"..",
		"..",
		"..",
		"fcaf_sources",
		"wallet_solution",
		"relying_party",
		"implementation",
	)
	for id, definition := range all {
		require.NotEmptyf(t, definition.Source.Path, "%s has no source path", id)
		sourcePath := filepath.Join(sourceRoot, filepath.Base(definition.Source.Path))
		require.FileExistsf(t, sourcePath, "%s source does not resolve", id)
	}
}

func generatedWalletRelyingPartyRoot() string {
	return "../../../config_templates/fcaf/wallet_solution/relying_party"
}

func loadDefinitionsFromDirectory(t *testing.T, root string) map[string]dsl.TestDefinition {
	t.Helper()
	definitions := map[string]dsl.TestDefinition{}
	require.NoError(
		t,
		filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".yaml") {
				return nil
			}
			definition, parseErr := dsl.ParseFile(path)
			if parseErr != nil {
				return parseErr
			}
			if _, duplicate := definitions[definition.ID]; duplicate {
				t.Fatalf("duplicate FCAF definition %s in %s", definition.ID, path)
			}
			definitions[definition.ID] = *definition
			return nil
		}),
	)
	return definitions
}
