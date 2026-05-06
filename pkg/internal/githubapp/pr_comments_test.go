// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build unit

package githubapp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitHubAPIURL(t *testing.T) {
	client := &Client{apiURL: "https://api.github.com"}

	require.Equal(
		t,
		"https://api.github.com/repos/ForkbombEu/eudi-app-android-wallet-ui/installation",
		client.githubAPIURL("repos", "ForkbombEu", "eudi-app-android-wallet-ui", "installation"),
	)
	require.Equal(
		t,
		"https://api.github.com/repos/ForkbombEu/eudi-app-android-wallet-ui/issues/1/comments?per_page=100",
		withQueryParam(
			client.githubAPIURL("repos", "ForkbombEu", "eudi-app-android-wallet-ui", "issues", "1", "comments"),
			"per_page",
			"100",
		),
	)
}
