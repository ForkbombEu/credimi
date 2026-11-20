// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package signedtoken

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

type grant struct {
	Name string `json:"name"`
}

func TestSignAndVerify(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	token, err := Sign(app, "label/a", grant{Name: "one"})
	require.NoError(t, err)
	var got grant
	require.NoError(t, Verify(app, "label/a", token, &got))
	assert.Equal(t, grant{Name: "one"}, got)

	other, err := Sign(app, "label/a", grant{Name: "two"})
	require.NoError(t, err)
	payload, signature, _ := strings.Cut(token, ".")
	otherPayload, _, _ := strings.Cut(other, ".")
	for name, tc := range map[string]struct{ label, token string }{
		"forged":           {label: "label/a", token: "forged.x"},
		"unsigned":         {label: "label/a", token: payload},
		"tampered payload": {label: "label/a", token: otherPayload + "." + signature},
		"another label":    {label: "label/b", token: token},
	} {
		t.Run(name, func(t *testing.T) {
			require.ErrorIs(t, Verify(app, tc.label, tc.token, &grant{}), ErrUnsigned)
		})
	}

	// Rotating the users token secret revokes every token.
	users, err := app.FindCollectionByNameOrId("users")
	require.NoError(t, err)
	users.AuthToken.Secret = strings.Repeat("r", 50)
	require.NoError(t, app.Save(users))
	require.ErrorIs(t, Verify(app, "label/a", token, &grant{}), ErrUnsigned)

	users.AuthToken.Secret = "short"
	require.NoError(t, app.SaveNoValidate(users))
	_, err = Sign(app, "label/a", grant{})
	require.ErrorContains(t, err, "too short")
}
