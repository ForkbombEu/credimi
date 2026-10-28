// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// GrantHeader carries a short-lived capability for one selected authoring APK.
const GrantHeader = "Credimi-Artifact-Grant"

type installerGrant struct {
	BuildRequest
	Namespace string `json:"namespace"`
	Filename  string `json:"filename"`
	APKSHA256 string `json:"apk_sha256"`
	Expires   int64  `json:"expires"`
}

func grantKey(app core.App) ([]byte, error) {
	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, fmt.Errorf("find grant signing collection: %w", err)
	}
	if len(collection.AuthToken.Secret) < 32 {
		return nil, fmt.Errorf("authoring grant signing secret unavailable")
	}
	key := hmac.New(sha256.New, []byte(collection.AuthToken.Secret))
	_, _ = io.WriteString(key, "credimi/action-author/installer/v1")
	return key.Sum(nil), nil
}

// IssueGrant authorizes one installer, never all files readable by a user. The
// short-lived capability stays out of model context and is checked again at download.
func IssueGrant(app core.App, selection Selection, digest string) (string, error) {
	key, err := grantKey(app)
	if err != nil {
		return "", err
	}
	claims := installerGrant{
		BuildRequest: BuildRequest{
			Actor:                   selection.Actor,
			WalletVersionIdentifier: selection.Identifier,
		},
		Namespace: selection.Organization.GetString(
			"canonified_name",
		),
		Filename:  selection.Version.GetString("android_installer"),
		APKSHA256: digest,
		Expires:   time.Now().Add(10 * time.Minute).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encode installer grant: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, key)
	_, _ = io.WriteString(mac, encoded)
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

type installerReader struct {
	io.ReadCloser
	filesystem io.Closer
}

func (r *installerReader) Close() error {
	err := r.ReadCloser.Close()
	fsErr := r.filesystem.Close()
	if err != nil {
		return err
	}
	return fsErr
}

// OpenInstaller reauthorizes the grant's actor and build before opening the selected APK.
func OpenInstaller(app core.App, token string) (io.ReadCloser, error) {
	if len(token) > 8192 {
		return nil, ErrForbidden
	}
	encoded, signature, ok := strings.Cut(token, ".")
	if !ok {
		return nil, ErrForbidden
	}
	key, err := grantKey(app)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = io.WriteString(mac, encoded)
	got, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil || !hmac.Equal(mac.Sum(nil), got) {
		return nil, ErrForbidden
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrForbidden
	}
	var claims installerGrant
	if json.Unmarshal(payload, &claims) != nil || claims.Expires <= time.Now().Unix() ||
		claims.Namespace == "" {
		return nil, ErrForbidden
	}
	selection, err := Select(app, claims.Namespace, claims.BuildRequest)
	if err != nil {
		return nil, err
	}
	if selection.Version.GetString("android_installer") != claims.Filename {
		return nil, ErrForbidden
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, fmt.Errorf("open installer filesystem: %w", err)
	}
	file, err := fsys.GetReader(selection.Version.BaseFilesPath() + "/" + claims.Filename)
	if err != nil {
		_ = fsys.Close()
		return nil, fmt.Errorf("open granted APK: %w", err)
	}
	return &installerReader{ReadCloser: file, filesystem: fsys}, nil
}
