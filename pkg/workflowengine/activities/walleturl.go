// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package activities is a package that provides activities for the workflow engine.
// This file includes activities for parsing wallet app urls.
package activities

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// ParseWalletURLActivityPayload is a struct that represents the input payload for the ParseWalletURLActivity.
type ParseWalletURLActivityPayload struct {
	URL string `json:"url" yaml:"url" validate:"required"`
}

type ParseWalletURLActivity struct {
	workflowengine.BaseActivity
}

func NewParseWalletURLActivity() *ParseWalletURLActivity {
	return &ParseWalletURLActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Parse app store URL (Apple or Google)",
		},
	}
}
func (a *ParseWalletURLActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute parses an application store URL from the workflow input payload
// and extracts an API input value and store type identifier.
// It supports Google Play Store and Apple App Store URLs:
//   - Google Play: the full URL is used as the API input, store_type = "google".
//   - Apple App Store: the numeric app ID is extracted from the path, store_type = "apple".
//
// If the URL is not supported, an error is returned.
func (a *ParseWalletURLActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var (
		result    workflowengine.ActivityResult
		apiInput  string
		storeType string
	)

	// Decode the input payload into a ParseWalletURLActivityPayload struct
	payload, err := workflowengine.DecodePayload[ParseWalletURLActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	parsed, err := url.Parse(TrimInput(payload.URL))
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ParseURLFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
		)
	}

	host := strings.ToLower(parsed.Host)

	switch {
	case strings.Contains(host, "play.google.com"):
		apiInput =
			payload.URL
		storeType = "google"

	case strings.Contains(host, "apps.apple.com"):
		re := regexp.MustCompile(`/id(\d+)`)
		matches := re.FindStringSubmatch(parsed.Path)
		if len(matches) == 0 {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
			return result, a.NewNonRetryableActivityError(
				errCode.Code,
				fmt.Sprintf("%s: 'url' is not a correct Apple store URL", errCode.Description),
				payload.URL,
			)
		}
		apiInput = matches[1]
		storeType = "apple"

	default:
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewNonRetryableActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'url' does not match a supported store type", errCode.Description),
			payload.URL,
		)
	}

	return workflowengine.ActivityResult{
		Output: map[string]any{
			"api_input":  apiInput,
			"store_type": storeType,
		},
	}, nil
}
