// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package recordsecrets

import (
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/pocketbase/pocketbase/core"
)

const secretsField = "secrets"

// collections hold a hidden "secrets" field, so PocketBase rejects filters and
// sorts on it and drops it from responses and non-superuser writes.
var collections = []string{"credentials", "use_cases_verifications"}

func RegisterHooks(app core.App) {
	for _, collection := range collections {
		app.OnRecordEnrich(collection).BindFunc(HandleSecretsEnrich)
		app.OnRecordCreateRequest(collection).BindFunc(HandleSecretsWrite)
		app.OnRecordUpdateRequest(collection).BindFunc(HandleSecretsWrite)
	}
}

// HandleSecretsEnrich shows secrets to superusers and to members of the owning
// organization only.
func HandleSecretsEnrich(e *core.RecordEnrichEvent) error {
	if canReadSecrets(e) {
		e.Record.Unhide(secretsField)
	} else {
		e.Record.Hide(secretsField)
	}
	return e.Next()
}

// HandleSecretsWrite keeps the submitted secrets of a write the collection
// rule already authorized.
func HandleSecretsWrite(e *core.RecordRequestEvent) error {
	if _, err := pbutils.LoadHiddenRequestFields(e, secretsField); err != nil {
		return e.BadRequestError("Failed to read the submitted data.", err)
	}
	return e.Next()
}

func canReadSecrets(e *core.RecordEnrichEvent) bool {
	if e.RequestInfo == nil || e.RequestInfo.Auth == nil {
		return false
	}
	if e.RequestInfo.HasSuperuserAuth() {
		return true
	}

	authOrgID, err := pbutils.GetUserOrganizationID(e.App, e.RequestInfo.Auth.Id)
	return err == nil && authOrgID == e.Record.GetString("owner")
}
