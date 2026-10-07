// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"encoding/base64"
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

func TestDecoderCacheEvictsLeastRecentlyUsedCredential(t *testing.T) {
	cache := NewDecoderCache(2)
	root := map[string]any{
		"a": testSDJWTWithEmail(t),
		"b": testSDJWTWithEmail(t) + " ",
		"c": testSDJWTWithEmail(t) + "  ",
	}
	extract := func(path string) *SDJWTPresentation {
		t.Helper()
		value, err := cache.Extract(root, path, "sdjwt.presentation")
		require.NoError(t, err)
		presentation, ok := value.(*SDJWTPresentation)
		require.True(t, ok)
		return presentation
	}

	a := extract("$.a")
	b := extract("$.b")
	require.Same(t, a, extract("$.a"), "reading a marks it as most recently used")
	extract("$.c")

	require.Same(t, a, extract("$.a"), "a survives because b was least recently used")
	require.NotSame(t, b, extract("$.b"), "b was evicted and decoded again")
}

func TestDecoderCacheClampsCapacityToOne(t *testing.T) {
	cache := NewDecoderCache(0)
	root := map[string]any{"a": testSDJWTWithEmail(t), "b": testSDJWTWithEmail(t) + " "}

	first, err := cache.Extract(root, "$.a", "sdjwt.presentation")
	require.NoError(t, err)
	again, err := cache.Extract(root, "$.a", "sdjwt.presentation")
	require.NoError(t, err)
	require.Same(t, first, again)

	_, err = cache.Extract(root, "$.b", "sdjwt.presentation")
	require.NoError(t, err)
	afterEviction, err := cache.Extract(root, "$.a", "sdjwt.presentation")
	require.NoError(t, err)
	require.NotSame(t, first, afterEviction)
}

func TestDecoderCacheKeysByDecoderAndSkipsNonCredentialDecoders(t *testing.T) {
	cache := NewDecoderCache(4)
	root := map[string]any{
		"json":     `{"ok":true}`,
		"vp_token": `{"query_0":["` + testSDJWTWithEmail(t) + `"]}`,
	}

	first, err := cache.Extract(root, "$.json", "json")
	require.NoError(t, err)
	first.(map[string]any)["ok"] = false
	second, err := cache.Extract(root, "$.json", "json")
	require.NoError(t, err)
	require.Equal(t, map[string]any{"ok": true}, second, "json results are never shared")

	single, err := cache.Extract(root, "$.vp_token", "sdjwt.vp_token_json")
	require.NoError(t, err)
	require.IsType(t, &SDJWTPresentation{}, single)
	all, err := cache.Extract(root, "$.vp_token", "sdjwt.vp_token_presentations_json")
	require.NoError(t, err)
	require.IsType(
		t,
		[]*SDJWTPresentation{},
		all,
		"non-credential decoders bypass the cache",
	)

	// mdoc.vp_token_json is also cacheable; on the same raw value it must decode
	// afresh (and fail on an SD-JWT token) instead of returning the cached
	// sdjwt.vp_token_json result.
	other, err := cache.Extract(root, "$.vp_token", "mdoc.vp_token_json")
	require.Error(t, err, "same input with another cacheable decoder is not a cache hit")
	require.Nil(t, other)

	mdoc := base64.RawURLEncoding.EncodeToString(
		testMDocDeviceResponse(t, map[string]any{"family_name": "Trotter"}),
	)
	mdocRoot := map[string]any{"vp_token": `{"pid_mdoc":["` + mdoc + `"]}`}
	mdocPresentation, err := cache.Extract(mdocRoot, "$.vp_token", "mdoc.vp_token_json")
	require.NoError(t, err)
	require.IsType(t, &MDocPresentation{}, mdocPresentation)
	sdjwtPresentation, err := cache.Extract(mdocRoot, "$.vp_token", "sdjwt.vp_token_json")
	require.Error(t, err, "same input with another cacheable decoder is not a cache hit")
	require.Nil(t, sdjwtPresentation)
}

func TestDecoderCacheDoesNotCacheFailures(t *testing.T) {
	cache := NewDecoderCache(2)
	root := map[string]any{"token": "not-an-sd-jwt"}

	_, err := cache.Extract(root, "$.token", "sdjwt.presentation")
	require.EqualError(t, err, "invalid SD-JWT presentation")
	_, err = cache.Extract(root, "$.token", "sdjwt.presentation")
	require.EqualError(t, err, "invalid SD-JWT presentation")
	require.Zero(t, cache.order.Len())

	_, err = cache.Extract(root, "$.missing", "sdjwt.presentation")
	require.EqualError(t, err, `missing key in path "missing"`)
}
