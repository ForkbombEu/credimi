// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
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

// HookTurnstileVerification validates Cloudflare Turnstile tokens on user registration and login.
// It reads the token from the X-Turnstile-Token HTTP header and verifies it with Cloudflare.
func HookTurnstileVerification(app *pocketbase.PocketBase) {
	app.OnRecordAuthWithPasswordRequest("users").BindFunc(
		func(e *core.RecordAuthWithPasswordRequestEvent) error {
			if err := requireTurnstile(e.RequestEvent); err != nil {
				return err
			}
			return e.Next()
		},
	)

	app.OnRecordAuthWithOAuth2Request("users").BindFunc(
		func(e *core.RecordAuthWithOAuth2RequestEvent) error {
			if err := requireTurnstile(e.RequestEvent); err != nil {
				return err
			}
			return e.Next()
		},
	)

	app.OnRecordCreateRequest("users").BindFunc(func(e *core.RecordRequestEvent) error {
		if isOAuth2RecordCreateRequest(e) {
			return e.Next()
		}

		if err := requireTurnstile(e.RequestEvent); err != nil {
			return err
		}

		return e.Next()
	})
}

func requireTurnstile(e *core.RequestEvent) error {
	token := e.Request.Header.Get("X-Turnstile-Token")
	if token == "" {
		return apis.NewBadRequestError("captcha verification failed", nil)
	}

	secret := utils.GetEnvironmentVariable("TURNSTILE_SECRET_KEY")
	if secret == "" {
		log.Printf("[turnstile] TURNSTILE_SECRET_KEY is not set, skipping verification")
		return nil
	}

	result, err := verifyTurnstileToken(token, secret)
	if err != nil {
		log.Printf("[turnstile] verification request failed: %v", err)
		return apis.NewBadRequestError("captcha verification failed", nil)
	}

	if !result.Success {
		log.Printf(
			"[turnstile] verification failed: codes=%v hostname=%s",
			result.ErrorCodes,
			result.Hostname,
		)
		return apis.NewBadRequestError("captcha verification failed", nil)
	}

	return nil
}

func isOAuth2RecordCreateRequest(e *core.RecordRequestEvent) bool {
	if e == nil || e.RequestEvent == nil {
		return false
	}

	requestInfo, err := e.RequestInfo()
	if err != nil {
		return false
	}

	return requestInfo.Context == core.RequestInfoContextOAuth2
}

func verifyTurnstileToken(token, secret string) (*turnstileVerifyResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data := url.Values{
		"secret":   {secret},
		"response": {token},
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Body = io.NopCloser(strings.NewReader(data.Encode()))

	client := &http.Client{}
	resp, err := client.Do(req)
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
