// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewsCreateIsSuperuserOnly(t *testing.T) {
	cases := []struct {
		name      string
		anonymous bool
		actor     rulesTestActor
		wantCode  int
	}{
		{name: "anonymous cannot publish news", anonymous: true, wantCode: http.StatusForbidden},
		{name: "user cannot publish news", actor: actorOrgOwner, wantCode: http.StatusForbidden},
		{name: "superuser can publish news", actor: actorSuperuser, wantCode: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newRulesTestApp(t)
			mux := buildRulesTestMux(t, app)
			token := ""
			if !tc.anonymous {
				token = rulesTestToken(t, app, tc.actor)
			}
			title := "DUMMY " + tc.name

			rec := serveRulesRequest(
				mux,
				http.MethodPost,
				"/api/collections/news/records",
				`{"title":"`+title+`","summary":"<p>dummy</p>","published":true,`+
					`"refer":"https://example.invalid/x"}`,
				token,
			)
			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())

			stored, err := app.FindAllRecords("news", dbx.HashExp{"title": title})
			require.NoError(t, err)
			wantStored := 0
			if tc.wantCode == http.StatusOK {
				wantStored = 1
			}
			assert.Len(t, stored, wantStored)

			// published news is public: anonymous readers see exactly what was stored
			list := serveRulesRequest(
				mux,
				http.MethodGet,
				"/api/collections/news/records?filter="+
					url.QueryEscape(`title="`+title+`"`),
				"",
				"",
			)
			require.Equal(t, http.StatusOK, list.Code, list.Body.String())
			var page struct {
				TotalItems int `json:"totalItems"`
			}
			require.NoError(t, json.Unmarshal(list.Body.Bytes(), &page))
			assert.Equal(t, wantStored, page.TotalItems)
		})
	}
}
