// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type turnstileVerifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
	Hostname   string   `json:"hostname"`
}

// HookTurnstileVerification validates Cloudflare Turnstile tokens on user registration.
// It reads the token from the X-Turnstile-Token HTTP header and verifies it with Cloudflare.
func HookTurnstileVerification(app *pocketbase.PocketBase) {
	app.OnRecordCreateRequest("users").BindFunc(func(e *core.RecordRequestEvent) error {
		token := e.Request.Header.Get("X-Turnstile-Token")
		if token == "" {
			return apis.NewBadRequestError("captcha verification failed", nil)
		}

		secret := utils.GetEnvironmentVariable("TURNSTILE_SECRET_KEY")
		if secret == "" {
			log.Printf("[turnstile] TURNSTILE_SECRET_KEY is not set, skipping verification")
			return e.Next()
		}

		result, err := verifyTurnstileToken(token, secret)
		if err != nil {
			log.Printf("[turnstile] verification request failed: %v", err)
			return apis.NewBadRequestError("captcha verification failed", nil)
		}

		if !result.Success {
			log.Printf("[turnstile] verification failed: codes=%v hostname=%s", result.ErrorCodes, result.Hostname)
			return apis.NewBadRequestError("captcha verification failed", nil)
		}

		return e.Next()
	})
}

func verifyTurnstileToken(token, secret string) (*turnstileVerifyResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
		"secret":   {secret},
		"response": {token},
	})
	if err != nil {
		return nil, fmt.Errorf("siteverify request failed: %w", err)
	}
	defer resp.Body.Close()

	var result turnstileVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode siteverify response: %w", err)
	}

	return &result, nil
}
