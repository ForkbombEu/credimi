// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package routes

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	_ "github.com/forkbombeu/credimi/migrations"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const (
	seedPasswordEnv          = "CREDIMI_SEED_SUPERUSER_PASSWORD"
	seedAdminEmail           = "admin@example.org"
	seedDefaultPassword      = "adminadmin"
	operatorAdminEmail       = "operator@example.org"
	operatorAdminSecret      = "operator-strong-password"
	defaultAdminLoginURL     = "/api/collections/_superusers/auth-with-password"
	removeSeedAdminMigration = "1790780000_remove_default_seed_admin.js"
)

var registerPBMigrationsOnce sync.Once

// registerPBMigrations loads pb_migrations into the process-wide
// core.AppMigrations list, as `credimi serve` does through jsvm, so every
// app bootstrapped afterwards applies them.
func registerPBMigrations(t testing.TB) {
	t.Helper()

	registerPBMigrationsOnce.Do(func() {
		app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
		require.NoError(t, jsvm.Register(app, jsvm.Config{
			MigrationsDir:     "../../pb_migrations",
			HooksDir:          "../../pb_hooks",
			HooksFilesPattern: "^$",
		}))
	})
}

// newDeployApp boots dataDir the way a (re)deployed `credimi serve` does:
// every pending migration is applied with the current environment.
func newDeployApp(t testing.TB, dataDir string) *tests.TestApp {
	t.Helper()

	registerPBMigrations(t)
	app, err := tests.NewTestApp(dataDir)
	require.NoError(t, err)
	return app
}

// newLegacyDataDir builds the volume of a deployment made before the seed was
// gated: the seed created admin@example.org with the default password and the
// cleanup migration is still pending. setup may alter it before the upgrade.
func newLegacyDataDir(t *testing.T, setup func(app core.App)) string {
	t.Helper()

	registerPBMigrations(t)
	dataDir := t.TempDir()
	t.Setenv(seedPasswordEnv, seedDefaultPassword)

	app := core.NewBaseApp(core.BaseAppConfig{DataDir: dataDir})
	require.NoError(t, app.Bootstrap())
	require.NoError(t, app.RunAllMigrations())
	_, err := app.DB().
		Delete(core.DefaultMigrationsTable, dbx.HashExp{"file": removeSeedAdminMigration}).
		Execute()
	require.NoError(t, err)
	if setup != nil {
		setup(app)
	}
	require.NoError(t, app.ResetBootstrapState())

	return dataDir
}

func saveSuperuser(t testing.TB, app core.App, email, password string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.SetEmail(email)
	record.SetPassword(password)
	require.NoError(t, app.Save(record))
}

func defaultAdminLogin(name string) tests.ApiScenario {
	return tests.ApiScenario{
		Name:   name,
		Method: http.MethodPost,
		URL:    defaultAdminLoginURL,
		Body: strings.NewReader(
			`{"identity":"` + seedAdminEmail + `","password":"` + seedDefaultPassword + `"}`,
		),
		Headers: map[string]string{"Content-Type": "application/json"},
	}
}

func rejectedDefaultAdminLogin(name string) tests.ApiScenario {
	scenario := defaultAdminLogin(name)
	scenario.ExpectedStatus = http.StatusBadRequest
	scenario.ExpectedContent = []string{`"data":{}`}
	scenario.NotExpectedContent = []string{`"token"`}
	return scenario
}

func acceptedDefaultAdminLogin(name string) tests.ApiScenario {
	scenario := defaultAdminLogin(name)
	scenario.ExpectedStatus = http.StatusOK
	scenario.ExpectedContent = []string{
		`"token"`,
		`"collectionName":"_superusers"`,
		`"email":"` + seedAdminEmail + `"`,
	}
	return scenario
}

func TestSeedAdminMigrationRequiresOperatorPassword(t *testing.T) {
	t.Run("fresh deploy without seed password has no superuser", func(t *testing.T) {
		t.Setenv(seedPasswordEnv, "")

		scenario := rejectedDefaultAdminLogin("default credential is rejected")
		scenario.TestAppFactory = func(t testing.TB) *tests.TestApp {
			return newDeployApp(t, t.TempDir())
		}
		scenario.AfterTestFunc = func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			total, err := app.CountRecords(core.CollectionNameSuperusers)
			require.NoError(t, err)
			require.Zero(t, total)
		}
		scenario.Test(t)
	})

	t.Run("dev and test seed password creates the admin", func(t *testing.T) {
		t.Setenv(seedPasswordEnv, seedDefaultPassword)

		scenario := acceptedDefaultAdminLogin("seeded admin logs in")
		scenario.TestAppFactory = func(t testing.TB) *tests.TestApp {
			return newDeployApp(t, t.TempDir())
		}
		scenario.Test(t)
	})
}

func TestRemoveDefaultSeedAdminMigration(t *testing.T) {
	t.Run("deletes the default admin when another superuser exists", func(t *testing.T) {
		dataDir := newLegacyDataDir(t, func(app core.App) {
			saveSuperuser(t, app, operatorAdminEmail, operatorAdminSecret)
		})
		t.Setenv(seedPasswordEnv, "")

		scenario := rejectedDefaultAdminLogin("default credential is gone")
		scenario.TestAppFactory = func(t testing.TB) *tests.TestApp {
			return newDeployApp(t, dataDir)
		}
		scenario.AfterTestFunc = func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			_, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, seedAdminEmail)
			require.Error(t, err)
			operator, err := app.FindAuthRecordByEmail(
				core.CollectionNameSuperusers,
				operatorAdminEmail,
			)
			require.NoError(t, err)
			require.True(t, operator.ValidatePassword(operatorAdminSecret))
		}
		scenario.Test(t)
	})

	t.Run("locks the default admin when it is the only superuser", func(t *testing.T) {
		dataDir := newLegacyDataDir(t, nil)
		t.Setenv(seedPasswordEnv, "")

		scenario := rejectedDefaultAdminLogin("default credential is rotated")
		scenario.TestAppFactory = func(t testing.TB) *tests.TestApp {
			return newDeployApp(t, dataDir)
		}
		scenario.AfterTestFunc = func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			admin, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, seedAdminEmail)
			require.NoError(t, err)
			require.False(t, admin.ValidatePassword(seedDefaultPassword))
		}
		scenario.Test(t)
	})

	t.Run("keeps an admin whose password was changed", func(t *testing.T) {
		dataDir := newLegacyDataDir(t, func(app core.App) {
			admin, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, seedAdminEmail)
			require.NoError(t, err)
			admin.SetPassword(operatorAdminSecret)
			require.NoError(t, app.Save(admin))
		})
		t.Setenv(seedPasswordEnv, "")

		scenario := rejectedDefaultAdminLogin("default credential never worked")
		scenario.TestAppFactory = func(t testing.TB) *tests.TestApp {
			return newDeployApp(t, dataDir)
		}
		scenario.AfterTestFunc = func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			admin, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, seedAdminEmail)
			require.NoError(t, err)
			require.True(t, admin.ValidatePassword(operatorAdminSecret))
		}
		scenario.Test(t)
	})

	t.Run("keeps the dev and test admin", func(t *testing.T) {
		dataDir := newLegacyDataDir(t, nil)

		scenario := acceptedDefaultAdminLogin("seeded admin still logs in")
		scenario.TestAppFactory = func(t testing.TB) *tests.TestApp {
			return newDeployApp(t, dataDir)
		}
		scenario.Test(t)
	})
}
