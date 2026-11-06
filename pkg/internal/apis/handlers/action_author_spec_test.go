// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

// A start body of an older client with spec fields queues a session without
// them: the agent sets each plan in the chat.
func TestActionAuthorStartIgnoresSpecFields(t *testing.T) {
	token := authoringToken(t, "userA")
	enqueued := stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:   "spec fields are ignored",
		Method: http.MethodPost,
		URL:    "/api/action-author/sessions",
		Body: strings.NewReader(`{"device_id":"` + authoringAdminRunner +
			`/android-emulator","wallet_version_identifier":"` + authoringAndroidVersion +
			`","slot":"onboarding","outcome":"rejection"}`),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"ticket_id"`},
		TestAppFactory:  setupActionAuthorApp,
	}).Test(t)
	require.NotEmpty(t, enqueued.TicketID)
	for _, field := range []string{"slot", "outcome", "wait_seconds", "prerequisites", "parameters"} {
		require.NotContains(t, enqueued.ActionAuthor, field)
	}
}
