// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DashboardCodeTTL is how long a dashboard sign-in code can be redeemed.
const DashboardCodeTTL = 60 * time.Second

// maxDashboardCodes bounds the pending codes this process keeps.
const maxDashboardCodes = 4096

// maxDashboardCodesPerUser bounds the pending codes of one user, so a single
// account cannot fill the process-wide bound.
const maxDashboardCodesPerUser = 5

// ErrTooManyDashboardCodes is returned when maxDashboardCodes unexpired codes
// are pending.
var ErrTooManyDashboardCodes = errors.New("too many pending dashboard sign-in codes")

// ErrTooManyUserDashboardCodes is returned when the user already has
// maxDashboardCodesPerUser unexpired codes pending.
var ErrTooManyUserDashboardCodes = errors.New(
	"too many pending dashboard sign-in codes for the user",
)

// DashboardCode is a pending sign-in to a runner's dashboard: the user with
// record UserID in AuthCollection may sign in to the dashboard of the runner
// with record ID RunnerID until Expires. ReturnTo is the dashboard callback
// the code was sent to; only that dashboard may redeem it.
type DashboardCode struct {
	RunnerID       string
	UserID         string
	AuthCollection string
	ReturnTo       string
	Expires        time.Time
}

// DashboardCodeStore holds pending codes by code. It lives in this process's
// memory: a restart drops them and users sign in again.
type DashboardCodeStore struct {
	mu    sync.Mutex
	codes map[string]DashboardCode
}

// NewDashboardCodeStore returns an empty store.
func NewDashboardCodeStore() *DashboardCodeStore {
	return &DashboardCodeStore{codes: map[string]DashboardCode{}}
}

// Issue stores c under a new random code and returns the code. Expired codes
// are dropped first; then a user with maxDashboardCodesPerUser pending codes
// gets ErrTooManyUserDashboardCodes and a store with maxDashboardCodes
// ErrTooManyDashboardCodes.
func (s *DashboardCodeStore) Issue(c DashboardCode, now time.Time) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate dashboard code: %w", err)
	}
	code := base64.RawURLEncoding.EncodeToString(raw)

	s.mu.Lock()
	defer s.mu.Unlock()
	pending := 0
	for k, v := range s.codes {
		if !now.Before(v.Expires) {
			delete(s.codes, k)
			continue
		}
		if v.AuthCollection == c.AuthCollection && v.UserID == c.UserID {
			pending++
		}
	}
	if pending >= maxDashboardCodesPerUser {
		return "", ErrTooManyUserDashboardCodes
	}
	if len(s.codes) >= maxDashboardCodes {
		return "", ErrTooManyDashboardCodes
	}
	s.codes[code] = c
	return code, nil
}

// Redeem removes code and returns its sign-in when it had not expired at now.
// A code can be redeemed once.
func (s *DashboardCodeStore) Redeem(code string, now time.Time) (DashboardCode, bool) {
	s.mu.Lock()
	c, ok := s.codes[code]
	delete(s.codes, code)
	s.mu.Unlock()
	if !ok || !now.Before(c.Expires) {
		return DashboardCode{}, false
	}
	return c, true
}
