// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecoderCacheReusesParsedCredential(t *testing.T) {
	cache := NewDecoderCache(2)
	root := map[string]any{
		"vp_token": `{"query_0":["eyJhbGciOiJub25lIn0.eyJ2Y3QiOiJ1cm46ZXVkaTpwaWQ6MSJ9.~"]}`,
	}

	first, err := cache.Extract(root, "$.vp_token", "sdjwt.vp_token_json")
	require.NoError(t, err)
	second, err := cache.Extract(root, "$.vp_token", "sdjwt.vp_token_json")
	require.NoError(t, err)

	firstPresentation, ok := first.(*SDJWTPresentation)
	require.True(t, ok)
	secondPresentation, ok := second.(*SDJWTPresentation)
	require.True(t, ok)
	require.Same(t, firstPresentation, secondPresentation)
}
