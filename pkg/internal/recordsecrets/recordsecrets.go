// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package recordsecrets

import (
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHooks(app core.App) {
	app.OnRecordEnrich("credentials").BindFunc(HandleSecretsEnrich)
	app.OnRecordEnrich("use_cases_verifications").BindFunc(HandleSecretsEnrich)
}

func HandleSecretsEnrich(e *core.RecordEnrichEvent) error {
	if e.RequestInfo == nil || e.RequestInfo.Auth == nil {
		e.Record.Hide("secrets")
		return e.Next()
	}
	if e.RequestInfo.HasSuperuserAuth() {
		return e.Next()
	}

	ownerID := e.Record.GetString("owner")
	authOrgID, err := pbutils.GetUserOrganizationID(e.App, e.RequestInfo.Auth.Id)
	if err != nil || authOrgID != ownerID {
		e.Record.Hide("secrets")
	}

	return e.Next()
}
