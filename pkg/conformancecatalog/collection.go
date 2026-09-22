// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// dropCollectionShell removes a leftover durable conformance_checks collection
// from data.db when present (legacy PR projection). Catalog data lives only in
// the process-private ephemeral store.
func dropCollectionShell(app core.App) error {
	collection := findCollectionShell(app)
	if collection == nil {
		return nil
	}
	if err := app.Delete(collection); err != nil {
		return fmt.Errorf("delete conformance_checks collection shell: %w", err)
	}
	return nil
}

func findCollectionShell(app core.App) *core.Collection {
	if c, err := app.FindCollectionByNameOrId(CollectionName); err == nil && c != nil {
		return c
	}
	if c, err := app.FindCollectionByNameOrId(CollectionID); err == nil && c != nil {
		return c
	}
	return nil
}
