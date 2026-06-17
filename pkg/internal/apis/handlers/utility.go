// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/core"
)

func authorizeOwnedOrPublishedRecord(
	record *core.Record,
	callerOrgID string,
	collectionName string,
	domain string,
) *apierror.APIError {
	if record.Collection().Name != collectionName {
		return apierror.New(
			http.StatusBadRequest,
			domain,
			domain+" identifier is invalid",
			domain+" identifier must resolve to a "+collectionName+" record",
		)
	}
	if record.GetString("owner") == callerOrgID || record.GetBool("published") {
		return nil
	}
	return apierror.New(
		http.StatusForbidden,
		domain,
		domain+" is not owned by caller or published",
		domain+" must belong to caller organization or be published",
	)
}
