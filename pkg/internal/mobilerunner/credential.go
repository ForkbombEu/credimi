// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// CredentialSecretEnvVar names the secret every runner credential is derived
// from. Rotating it invalidates every runner credential until the runners
// register again.
const CredentialSecretEnvVar = "CREDIMI_RUNNER_CREDENTIAL_SECRET"

// ErrCredentialSecretMissing reports that CredentialSecretEnvVar is not set.
var ErrCredentialSecretMissing = errors.New(CredentialSecretEnvVar + " is not set")

// Credential returns the key Credimi presents to runner as Credimi-Api-Key:
// the lowercase hex HMAC-SHA256, keyed by CredentialSecretEnvVar, of
// "<record id>:<credential_generation>". Each runner gets its own value, and
// bumping credential_generation rotates it. Credimi never sends its internal
// admin key to a runner.
func Credential(runner *core.Record) (string, error) {
	secret := strings.TrimSpace(os.Getenv(CredentialSecretEnvVar))
	if secret == "" {
		return "", ErrCredentialSecretMissing
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(runner.Id + ":" + strconv.Itoa(runner.GetInt("credential_generation"))))

	return hex.EncodeToString(mac.Sum(nil)), nil
}
