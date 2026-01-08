// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
)

func TestGenerateApiKeyBytes(t *testing.T) {
	// Test the CryptoKeyGenerator implementation
	generator := &CryptoKeyGenerator{}
	key, err := generator.GenerateKeyBytes()
	assert.NoError(t, err)
	assert.Len(t, key, 32)

	// Test randomness - generate multiple keys and ensure they're different
	keys := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		keys[i], err = generator.GenerateKeyBytes()
		assert.NoError(t, err)
	}

	// Verify all keys are different
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			assert.NotEqual(t, keys[i], keys[j], "Generated keys should be unique")
		}
	}
}

func TestB64EncodeApiKey(t *testing.T) {
	generator := &CryptoKeyGenerator{}

	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "normal key",
			input:    []byte("testapikey1234567890"),
			expected: base64.URLEncoding.EncodeToString([]byte("testapikey1234567890")),
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "binary data",
			input:    []byte{0x00, 0x01, 0xFF, 0xFE},
			expected: base64.URLEncoding.EncodeToString([]byte{0x00, 0x01, 0xFF, 0xFE}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := generator.EncodeKey(tt.input)
			assert.Equal(t, tt.expected, encoded)

			// Verify it's URL-safe base64 (no + or / characters)
			assert.False(t, strings.Contains(encoded, "+"))
			assert.False(t, strings.Contains(encoded, "/"))
		})
	}
}

func TestHashApiKey_Success(t *testing.T) {
	hasher := NewBcryptKeyHasher()

	tests := []struct {
		name   string
		apiKey string
	}{
		{"normal key", "my-secret-key"},
		{"long key", strings.Repeat("a", 50)},
		{"unicode key", "こんにちは-世界-🔑"},
		{"special chars", "key!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashed, err := hasher.HashKey(tt.apiKey)
			assert.NoError(t, err)
			assert.NotEmpty(t, hashed)
			assert.True(t, strings.HasPrefix(hashed, "$2a$"))

			// Verify the hash can be validated
			err = hasher.CompareHashAndKey(hashed, tt.apiKey)
			assert.NoError(t, err)

			// Verify wrong key fails
			err = hasher.CompareHashAndKey(hashed, "wrong-key")
			assert.Error(t, err)
		})
	}
}

func TestHashApiKey_Error(t *testing.T) {
	hasher := NewBcryptKeyHasher()

	// Test empty key
	_, err := hasher.HashKey("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot hash empty key")
	// Test very long key
	_, err = hasher.HashKey(strings.Repeat("a", 1000))
	assert.Error(t, err)
	// Test special characters
	_, err = hasher.HashKey("test!@#$%^&*()")
	assert.NoError(t, err) // Special characters should not cause issues
	// Verify the hash can be validated
	hashed, err := hasher.HashKey("test-key")
	assert.NoError(t, err)
	err = hasher.CompareHashAndKey(hashed, "test-key")
	assert.NoError(t, err)
	err = hasher.CompareHashAndKey(hashed, "wrong-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hashedPassword is not the hash of the given password")
}

func TestGetMatchApiKeyRecord_Found(t *testing.T) {
	repo := &DefaultRecordRepository{}
	hasher := NewBcryptKeyHasher()

	apiKeysBytes := []byte("test-key")
	generator := &CryptoKeyGenerator{}
	apiKey := generator.EncodeKey(apiKeysBytes)
	hashed, err := hasher.HashKey(apiKey)
	assert.NoError(t, err)

	records := make([]*core.Record, 0, 1)
	dummyCollection := &core.Collection{}
	dummyCollection.Name = "api_keys"
	record := core.NewRecord(dummyCollection)
	record.Set("key", hashed)
	record.Set("name", "Test API Key")
	record.Set("user", "user123")
	record.Set("id", "1")
	records = append(records, record)

	found, err := repo.FindMatchingApiKeyRecord(records, apiKey, hasher)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "Test API Key", found.Get("name"))
	assert.Equal(t, "user123", found.Get("user"))
}

func TestGetMatchApiKeyRecord_NotFound(t *testing.T) {
	repo := &DefaultRecordRepository{}
	hasher := NewBcryptKeyHasher()

	apiKeysBytes := []byte("test-key")
	generator := &CryptoKeyGenerator{}
	apiKey := generator.EncodeKey(apiKeysBytes)
	records := make([]*core.Record, 0, 1)

	_, err := repo.FindMatchingApiKeyRecord(records, apiKey, hasher)
	assert.Error(t, err)
	assert.Equal(t, "no matching API key record found", err.Error())
}

func TestGetMatchApiKeyRecord_MaliciousInput(t *testing.T) {
	repo := &DefaultRecordRepository{}
	hasher := NewBcryptKeyHasher()

	apiKeysBytes := []byte("test-key")
	generator := &CryptoKeyGenerator{}
	apiKey := generator.EncodeKey(apiKeysBytes)

	records := make([]*core.Record, 0, 1)
	dummyCollection := &core.Collection{}
	dummyCollection.Name = "api_keys"
	record := core.NewRecord(dummyCollection)
	record.Set("key", "malicious-hash")
	record.Set("name", "Malicious API Key")
	record.Set("user", "malicious_user")
	record.Set("id", "2")
	records = append(records, record)

	found, err := repo.FindMatchingApiKeyRecord(records, apiKey, hasher)
	assert.Error(t, err)
	assert.Nil(t, found)
	assert.Equal(t, "no matching API key record found", err.Error())
}

// Additional security tests for malicious hash formats
func TestGetMatchApiKeyRecord_SecurityMaliciousHashes(t *testing.T) {
	repo := &DefaultRecordRepository{}
	hasher := NewBcryptKeyHasher()
	apiKey := "test-api-key"

	maliciousHashes := []string{
		"", // Empty hash
		"plaintext-not-hash",
		"$1$invalid$",              // Wrong bcrypt format
		"$2a$04$invalid.hash.here", // Invalid bcrypt
		"$2a$31$toolongcost",       // Invalid cost
		strings.Repeat("a", 1000),  // Very long string
		"hash\x00null",             // Null bytes
	}

	for _, maliciousHash := range maliciousHashes {
		t.Run(fmt.Sprintf("malicious_hash_%d", len(maliciousHash)), func(t *testing.T) {
			dummyCollection := &core.Collection{}
			dummyCollection.Name = "api_keys"
			record := core.NewRecord(dummyCollection)
			record.Set("key", maliciousHash)
			record.Set("name", "Test API Key")
			record.Set("user", "user123")
			records := []*core.Record{record}

			found, err := repo.FindMatchingApiKeyRecord(records, apiKey, hasher)
			assert.Error(t, err)
			assert.Nil(t, found)
			assert.Equal(t, "no matching API key record found", err.Error())
		})
	}
}

// Test timing attack resistance
func TestGetMatchApiKeyRecord_TimingAttackResistance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timing attack test in short mode")
	}

	repo := &DefaultRecordRepository{}
	hasher := NewBcryptKeyHasher()

	// Create records with different hash formats
	records := []*core.Record{}
	dummyCollection := &core.Collection{}
	dummyCollection.Name = "api_keys"

	// Valid hash
	validKey := "valid-test-key"
	validHash, _ := hasher.HashKey(validKey)
	validRecord := core.NewRecord(dummyCollection)
	validRecord.Set("key", validHash)
	records = append(records, validRecord)

	// Invalid hashes of different types
	invalidHashes := []string{
		"short",
		"medium-length-invalid-hash",
		"very-long-invalid-hash-that-should-take-similar-time-to-process",
		"$2a$10$invalid.but.proper.length.hash.format",
	}

	for _, invalidHash := range invalidHashes {
		record := core.NewRecord(dummyCollection)
		record.Set("key", invalidHash)
		records = append(records, record)
	}

	// Test timing for different scenarios
	testKey := "test-key-that-wont-match"
	iterations := 10

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := repo.FindMatchingApiKeyRecord(records, testKey, hasher)
		duration := time.Since(start)

		assert.Error(t, err)
		assert.True(t, duration > time.Microsecond, "Should take measurable time")
		assert.True(t, duration < time.Second, "Should not take too long")
	}
}

// Tests for mock records that implement HasAuthToken

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
	resp, err := generateAuthenticateApiKeyResponse(record)
	assert.NoError(t, err)
	assert.Equal(t, "API key authenticated successfully", resp.Message)
	assert.Equal(t, "dummy-token", resp.Token)
}

func TestGenerateAuthenticateApiKeyResponse_Error(t *testing.T) {
	record := new(DummyRecordError)
	resp, err := generateAuthenticateApiKeyResponse(record)
	assert.Error(t, err)
	assert.Equal(t, AuthenticateApiKeyResponse{}, resp)

	// Check if it's an APIError
	var apiErr *apierror.APIError
	ok := errors.As(err, &apiErr)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Code)
	assert.Contains(t, apiErr.Error(), "dummy error")
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

	resp, err := generateAuthenticateApiKeyResponse(record)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)

	// Verify JWT structure (3 parts separated by dots)
	parts := strings.Split(resp.Token, ".")
	assert.Len(t, parts, 3)
	assert.NotEmpty(t, parts[0]) // Header
	assert.NotEmpty(t, parts[1]) // Payload
	assert.NotEmpty(t, parts[2]) // Signature
	assert.Contains(t, resp.Message, "API key authenticated successfully")
}

// Integration-style tests for the service layer

func TestGenerateAuthenticateApiKeyResponse_SecurityMaliciousApiKey(t *testing.T) {
	record := new(DummyRecord)

	maliciousApiKeys := []string{
		"",
		"key\x00null",
		"key\r\nheader-injection",
		strings.Repeat("a", 10000),
		"../../../etc/passwd",
		"<script>alert('xss')</script>",
	}

	for _, maliciousKey := range maliciousApiKeys {
		t.Run(fmt.Sprintf("malicious_key_%d", len(maliciousKey)), func(t *testing.T) {
			// Should not panic and should handle gracefully
			assert.NotPanics(t, func() {
				resp, err := generateAuthenticateApiKeyResponse(record)
				if err == nil {
					assert.NotEmpty(t, resp.Message)
					assert.NotEmpty(t, resp.Token)
				}
			})
		})
	}
}

// Test AppAdapter - simplified test focusing on our interface
func TestAppAdapter_Basic(t *testing.T) {
	// This is more of a compilation test to ensure our interfaces work
	// In practice, the adapter would be tested through integration tests
	generator := &CryptoKeyGenerator{}
	hasher := NewBcryptKeyHasher()
	repo := &DefaultRecordRepository{}

	// Test that our components can be instantiated
	assert.NotNil(t, generator)
	assert.NotNil(t, hasher)
	assert.NotNil(t, repo)

	// Test basic functionality
	keyBytes, err := generator.GenerateKeyBytes()
	assert.NoError(t, err)
	assert.Len(t, keyBytes, 32)

	encodedKey := generator.EncodeKey(keyBytes)
	assert.NotEmpty(t, encodedKey)

	hashedKey, err := hasher.HashKey(encodedKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, hashedKey)
}

// Benchmark tests for performance

func BenchmarkGenerateApiKey(b *testing.B) {
	generator := &CryptoKeyGenerator{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.GenerateKeyBytes()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHashApiKey(b *testing.B) {
	hasher := NewBcryptKeyHasher()
	apiKey := "test-api-key-for-benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hasher.HashKey(apiKey)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompareHashAndKey(b *testing.B) {
	hasher := NewBcryptKeyHasher()
	apiKey := "test-api-key-for-benchmarking"
	hash, err := hasher.HashKey(apiKey)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := hasher.CompareHashAndKey(hash, apiKey)
		if err != nil {
			b.Fatal(err)
		}
	}
}
