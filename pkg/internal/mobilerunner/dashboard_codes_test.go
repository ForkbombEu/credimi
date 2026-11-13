// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dashboardTestNow = time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

func dashboardTestCode(userID string, now time.Time) DashboardCode {
	return DashboardCode{
		RunnerID:       "runner",
		UserID:         userID,
		AuthCollection: "users",
		ReturnTo:       "http://dashboard/auth/credimi/callback",
		Expires:        now.Add(DashboardCodeTTL),
	}
}

func TestDashboardCodeStoreRedeem(t *testing.T) {
	t.Run("a code is redeemed once", func(t *testing.T) {
		s := NewDashboardCodeStore()
		want := dashboardTestCode("u1", dashboardTestNow)
		code, err := s.Issue(want, dashboardTestNow)
		require.NoError(t, err)

		got, ok := s.Redeem(code, dashboardTestNow.Add(time.Second))
		require.True(t, ok)
		assert.Equal(t, want, got)
		assert.Empty(t, s.codes)

		_, ok = s.Redeem(code, dashboardTestNow.Add(time.Second))
		assert.False(t, ok)
	})

	t.Run("an expired code is refused and removed", func(t *testing.T) {
		s := NewDashboardCodeStore()
		code, err := s.Issue(dashboardTestCode("u1", dashboardTestNow), dashboardTestNow)
		require.NoError(t, err)

		_, ok := s.Redeem(code, dashboardTestNow.Add(DashboardCodeTTL))
		assert.False(t, ok)
		assert.Empty(t, s.codes)
	})

	t.Run("an unknown code is refused", func(t *testing.T) {
		s := NewDashboardCodeStore()
		_, ok := s.Redeem("nope", dashboardTestNow)
		assert.False(t, ok)
	})
}

func TestDashboardCodeStorePerUserCap(t *testing.T) {
	s := NewDashboardCodeStore()
	for range maxDashboardCodesPerUser {
		_, err := s.Issue(dashboardTestCode("u1", dashboardTestNow), dashboardTestNow)
		require.NoError(t, err)
	}

	_, err := s.Issue(dashboardTestCode("u1", dashboardTestNow), dashboardTestNow)
	require.ErrorIs(t, err, ErrTooManyUserDashboardCodes)

	t.Run("other users and collections are not affected", func(t *testing.T) {
		_, err := s.Issue(dashboardTestCode("u2", dashboardTestNow), dashboardTestNow)
		require.NoError(t, err)
		superuser := dashboardTestCode("u1", dashboardTestNow)
		superuser.AuthCollection = "_superusers"
		_, err = s.Issue(superuser, dashboardTestNow)
		require.NoError(t, err)
	})

	t.Run("expired codes no longer count", func(t *testing.T) {
		later := dashboardTestNow.Add(DashboardCodeTTL)
		_, err := s.Issue(dashboardTestCode("u1", later), later)
		require.NoError(t, err)
	})
}

func TestDashboardCodeStoreGlobalCap(t *testing.T) {
	fill := func(s *DashboardCodeStore, expires time.Time) {
		for i := range maxDashboardCodes {
			c := dashboardTestCode("user-"+strconv.Itoa(i), dashboardTestNow)
			c.Expires = expires
			s.codes["code-"+strconv.Itoa(i)] = c
		}
	}

	t.Run("a full store refuses new codes", func(t *testing.T) {
		s := NewDashboardCodeStore()
		fill(s, dashboardTestNow.Add(DashboardCodeTTL))

		_, err := s.Issue(dashboardTestCode("someone", dashboardTestNow), dashboardTestNow)
		require.ErrorIs(t, err, ErrTooManyDashboardCodes)
		assert.Len(t, s.codes, maxDashboardCodes)
	})

	t.Run("expired codes are pruned before the bound is checked", func(t *testing.T) {
		s := NewDashboardCodeStore()
		fill(s, dashboardTestNow)

		code, err := s.Issue(dashboardTestCode("someone", dashboardTestNow), dashboardTestNow)
		require.NoError(t, err)
		assert.Len(t, s.codes, 1)
		assert.Contains(t, s.codes, code)
	})
}
