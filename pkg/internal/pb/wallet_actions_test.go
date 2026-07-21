// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestWalletActionHooksRequireInstallCategoryForPlayStoreLink(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	RegisterWalletActionHooks(app)

	record := core.NewRecord(core.NewBaseCollection(walletActionsCollectionName))
	record.Set(
		"code",
		"appId: com.sphereon.ssi.wallet\n---\n- openLink: \"market://details?id=com.sphereon.ssi.wallet\"",
	)
	record.Set("category", "onboarding")

	event := &core.RecordEvent{App: app}
	event.Record = record
	err := app.OnRecordCreate(walletActionsCollectionName).Trigger(
		event,
		func(_ *core.RecordEvent) error { return nil },
	)
	require.EqualError(t, err, validationWalletActionMarketLinkRequiresInstallApp)
}

func TestWalletActionHooksAllowInstallCategoryForPlayStoreLink(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	RegisterWalletActionHooks(app)

	record := core.NewRecord(core.NewBaseCollection(walletActionsCollectionName))
	record.Set("code", "- openLink: \"market://details?id=com.sphereon.ssi.wallet\"")
	record.Set("category", "install-app")

	event := &core.RecordEvent{App: app}
	event.Record = record
	err := app.OnRecordCreate(walletActionsCollectionName).Trigger(
		event,
		func(_ *core.RecordEvent) error { return nil },
	)
	require.NoError(t, err)
}
