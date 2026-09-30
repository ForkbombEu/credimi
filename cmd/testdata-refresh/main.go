// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command testdata-refresh regenerates test_pb_data/data.db with all pending
// Go and pb_migrations JS migrations applied, because tests.NewTestApp loads
// neither jsvm nor pending migrations. `make test` runs it before the suite;
// it leaves data.db untouched when nothing is pending.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/tests"
)

func main() {
	target := "test_pb_data"
	if len(os.Args) > 1 {
		target = os.Args[1]
	}

	app, err := tests.NewTestApp(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create test app:", err)
		os.Exit(1)
	}

	err = refresh(app, target)
	app.Cleanup()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func refresh(app *tests.TestApp, target string) error {
	// Load pb_migrations only; the hooks pattern matches no file, so pb_hooks
	// never run while migrations save collections.
	if err := jsvm.Register(app, jsvm.Config{
		MigrationsDir:     "pb_migrations",
		HooksDir:          "pb_hooks",
		HooksFilesPattern: "^$",
	}); err != nil {
		return fmt.Errorf("failed to register js migrations: %w", err)
	}

	before, err := countAppliedMigrations(app)
	if err != nil {
		return fmt.Errorf("failed to count migrations: %w", err)
	}
	if err := app.RunAllMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	after, err := countAppliedMigrations(app)
	if err != nil {
		return fmt.Errorf("failed to count migrations: %w", err)
	}
	dbPath := filepath.Join(target, "data.db")
	if after == before {
		fmt.Println("up to date", dbPath)
		return nil
	}

	// Close cleanly so SQLite checkpoints the WAL into data.db before we copy
	// it back into the source directory.
	if err := app.ResetBootstrapState(); err != nil {
		return fmt.Errorf("failed to close app cleanly: %w", err)
	}
	in, err := os.ReadFile(filepath.Join(app.DataDir(), "data.db"))
	if err != nil {
		return fmt.Errorf("failed to read migrated db: %w", err)
	}
	if err := os.WriteFile(dbPath, in, 0o644); err != nil {
		return fmt.Errorf("failed to write migrated db: %w", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(dbPath + suffix)
	}

	fmt.Printf("refreshed %s (%d migrations applied)\n", dbPath, after-before)
	return nil
}

func countAppliedMigrations(app core.App) (int, error) {
	var count int
	err := app.DB().Select("count(*)").From(core.DefaultMigrationsTable).Row(&count)
	return count, err
}
