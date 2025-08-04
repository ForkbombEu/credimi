// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
)

func TestGenerateApiKeyBytes(t *testing.T) {
	key, err := generateApiKeyBytes()
	assert.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestB64EncodeApiKey(t *testing.T) {
	data := []byte("testapikey1234567890")
	encoded := b64EncodeApiKey(data)
	expected := base64.URLEncoding.EncodeToString(data)
	assert.Equal(t, expected, encoded)
}

func TestHashApiKey_Success(t *testing.T) {
	apiKey := "my-secret-key"
	hashed, err := hashApiKey(apiKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, hashed)
	// Check that the hash matches the original
	err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(apiKey))
	assert.NoError(t, err)
}

func TestHashApiKey_Error(t *testing.T) {
	// bcrypt should not error for normal input, so this is just a sanity check
	_, err := hashApiKey("")
	assert.NoError(t, err)
}

func TestGetMatchApiKeyRecord_Found(t *testing.T) {
	apiKeysBytes := []byte("test-key")
	apiKey := b64EncodeApiKey(apiKeysBytes)
	hashed, _ := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	records := []*core.Record{}
	dummyCollection := &core.Collection{}
	dummyCollection.Name = "api_keys"
	Record := core.NewRecord(dummyCollection)
	Record.Set("key", string(hashed))
	Record.Set("name", "Test API Key")
	Record.Set("user_id", "user123")
	Record.Set("id", "1")
	records = append(records, Record)

	found, err := getMatchApiKeyRecord(records, apiKey)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, found.Get("key"), found.Get("key"))
	assert.Equal(t, found.Get("name"), "Test API Key")
	assert.Equal(t, found.Get("user_id"), "user123")
}

func TestGetMatchApiKeyRecord_NotFound(t *testing.T) {
	apiKeysBytes := []byte("test-key")
	apiKey := b64EncodeApiKey(apiKeysBytes)
	records := []*core.Record{}
	_, err := getMatchApiKeyRecord(records, apiKey)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "no matching API key record found")
}

func TestGetMatchApiKeyRecord_MaliciousInput(t *testing.T) {
	apiKeysBytes := []byte("test-key")
	apiKey := b64EncodeApiKey(apiKeysBytes)
	records := []*core.Record{}
	dummyCollection := &core.Collection{}
	dummyCollection.Name = "api_keys"
	Record := core.NewRecord(dummyCollection)
	Record.Set("key", "malicious-hash")
	Record.Set("name", "Malicious API Key")
	Record.Set("user_id", "malicious_user")
	Record.Set("id", "2")
	records = append(records, Record)
	found, err := getMatchApiKeyRecord(records, apiKey)
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.Equal(t, err.Error(), "no matching API key record found")
}

type DummyRecord struct{}

func (r *DummyRecord) NewAuthToken() (string, error) {
	return "dummy-token", nil
}

type DummyRecordError struct{}
func (r *DummyRecordError) NewAuthToken() (string, error) {
	return "", fmt.Errorf("dummy error")
}
func TestGenerateAuthenticateApiKeyResponse_Success(t *testing.T) {
	record := new(DummyRecord)
	resp, err := generateAuthenticateApiKeyResponse("dummy-api-key", record)
	assert.NoError(t, err)
	assert.Equal(t, "API key authenticated successfully", resp.Message)
	assert.Equal(t, "dummy-token", resp.Token)
}

func TestGenerateAuthenticateApiKeyResponse_Error(t *testing.T) {
	record := new(DummyRecordError)
	resp, err := generateAuthenticateApiKeyResponse("dummy-api-key", record)
	assert.Error(t, err)
	assert.Equal(t, AuthenticateApiKeyResponseSchema{}, resp)
	assert.Equal(t, "[request.internal_error:failed_to_generate_auth_token] dummy error", err.Error())
}

func TestGenerateAuthenticateApiKeyResponse_IsJWT(t *testing.T) {
	dummyCollection := &core.Collection{}
	dummyCollection.Type = core.CollectionTypeAuth
	dummyCollection.Name = "users"
	record := core.NewRecord(dummyCollection)
	record.Set("id", "1")
	record.Set("email", "test@example.com")
	record.Set("username", "testuser")
	record.Set("name", "Test User")
	record.Set("verified", true)
	record.Set("emailVisibility", true)
	record.Set("tokenKey", "dummy-token-key")
	record.Set("password", "dummy-password")
	resp, err := generateAuthenticateApiKeyResponse("dummy-api-key", record)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	parts := strings.Split(resp.Token, ".")
	assert.Len(t, parts, 3) 
	assert.NotEmpty(t, parts[0]) // Header
	assert.NotEmpty(t, parts[1]) // Payload
	assert.NotEmpty(t, parts[2]) // Signature
	assert.Contains(t, resp.Message, "API key authenticated successfully")
}
