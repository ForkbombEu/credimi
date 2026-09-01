// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package webpush

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	webpushgo "github.com/SherClockHolmes/webpush-go"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// EnvVAPIDPublicKey overrides the stored VAPID public key.
	EnvVAPIDPublicKey = "CREDIMI_WEB_PUSH_VAPID_PUBLIC_KEY"
	// EnvVAPIDPrivateKey overrides the stored VAPID private key.
	EnvVAPIDPrivateKey = "CREDIMI_WEB_PUSH_VAPID_PRIVATE_KEY"

	webPushSettingsCollection = "web_push_settings"
)

// GetVAPIDKeyPair returns the Web Push VAPID key pair, generating and
// persisting a new one on first use. When both environment variables are set
// they take precedence over the stored key pair.
func GetVAPIDKeyPair(app core.App) (publicKey string, privateKey string, err error) {
	envPublicKey := strings.TrimSpace(os.Getenv(EnvVAPIDPublicKey))
	envPrivateKey := strings.TrimSpace(os.Getenv(EnvVAPIDPrivateKey))
	switch {
	case envPublicKey != "" && envPrivateKey != "":
		return envPublicKey, envPrivateKey, nil
	case envPublicKey != "" || envPrivateKey != "":
		return "", "", fmt.Errorf(
			"%s and %s must both be set",
			EnvVAPIDPublicKey,
			EnvVAPIDPrivateKey,
		)
	}

	record, err := app.FindFirstRecordByFilter(webPushSettingsCollection, "")
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("failed to load web push settings: %w", err)
		}

		generatedPrivateKey, generatedPublicKey, err := webpushgo.GenerateVAPIDKeys()
		if err != nil {
			return "", "", fmt.Errorf("failed to generate VAPID keys: %w", err)
		}
		if err := saveVAPIDKeyPair(app, generatedPublicKey, generatedPrivateKey); err != nil {
			return "", "", err
		}
		return generatedPublicKey, generatedPrivateKey, nil
	}

	publicKey = record.GetString("vapid_public_key")
	privateKey = record.GetString("vapid_private_key")
	if publicKey == "" || privateKey == "" {
		return "", "", fmt.Errorf("stored VAPID key pair is incomplete")
	}
	return publicKey, privateKey, nil
}

func saveVAPIDKeyPair(app core.App, publicKey string, privateKey string) error {
	collection, err := app.FindCollectionByNameOrId(webPushSettingsCollection)
	if err != nil {
		return fmt.Errorf("failed to get web push settings collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("vapid_public_key", publicKey)
	record.Set("vapid_private_key", privateKey)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("failed to store VAPID key pair: %w", err)
	}
	return nil
}
