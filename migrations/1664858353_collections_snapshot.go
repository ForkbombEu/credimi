// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package migrations

import (
	_ "embed"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

//go:embed pb_schema.json
var jsonData string

// Auto generated migration with the most recent collections configuration.
func init() {
	m.Register(func(app core.App) error {
		return app.ImportCollectionsByMarshaledJSON([]byte(jsonData), false)
	}, func(_ core.App) error {
		// no revert since the configuration on the environment, on which
		// the migration was executed, could have changed via the UI/API
		return nil
	})
}
