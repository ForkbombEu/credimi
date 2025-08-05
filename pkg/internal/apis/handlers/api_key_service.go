// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/crypto/bcrypt"
)

type ApiKeyService struct {
	app              App
	keyGenerator     KeyGenerator
	keyHasher        KeyHasher
	recordRepository RecordRepository
}

type App interface {
	FindCollectionByNameOrId(collectionKey string) (*core.Collection, error)
	Save(record *core.Record) error
	FindRecordsByFilter(collectionNameOrId, filter, sort string, limit, offset int) ([]*core.Record, error)
	FindRecordById(collectionNameOrId, recordId string) (*core.Record, error)
}

type AppAdapter struct {
	coreApp core.App
}

func NewAppAdapter(coreApp core.App) *AppAdapter {
	return &AppAdapter{coreApp: coreApp}
}

func (a *AppAdapter) FindCollectionByNameOrId(collectionKey string) (*core.Collection, error) {
	return a.coreApp.FindCollectionByNameOrId(collectionKey)
}

func (a *AppAdapter) Save(record *core.Record) error {
	return a.coreApp.Save(record)
}

func (a *AppAdapter) FindRecordsByFilter(
	collectionNameOrId,
	filter,
	sort string,
	limit,
	offset int,
) ([]*core.Record, error) {
	return a.coreApp.FindRecordsByFilter(collectionNameOrId, filter, sort, limit, offset)
}

func (a *AppAdapter) FindRecordById(collectionNameOrId, recordId string) (*core.Record, error) {
	return a.coreApp.FindRecordById(collectionNameOrId, recordId)
}

type KeyGenerator interface {
	GenerateKeyBytes() ([]byte, error)
	EncodeKey(keyBytes []byte) string
}

type KeyHasher interface {
	HashKey(key string) (string, error)
	CompareHashAndKey(hashedKey, key string) error
}

type RecordRepository interface {
	FindMatchingApiKeyRecord(records []*core.Record, apiKey string, hasher KeyHasher) (*core.Record, error)
}

type CryptoKeyGenerator struct{}

func (g *CryptoKeyGenerator) GenerateKeyBytes() ([]byte, error) {
	apiKeyBytes := make([]byte, 32)
	if _, err := rand.Read(apiKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate API key bytes: %w", err)
	}
	return apiKeyBytes, nil
}

func (g *CryptoKeyGenerator) EncodeKey(keyBytes []byte) string {
	return base64.URLEncoding.EncodeToString(keyBytes)
}

type BcryptKeyHasher struct {
	Cost int
}

func NewBcryptKeyHasher() *BcryptKeyHasher {
	return &BcryptKeyHasher{Cost: bcrypt.DefaultCost}
}

func (h *BcryptKeyHasher) HashKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("cannot hash empty key")
	}
	hashedKey, err := bcrypt.GenerateFromPassword([]byte(key), h.Cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hashedKey), nil
}

func (h *BcryptKeyHasher) CompareHashAndKey(hashedKey, key string) error {
	if hashedKey == "" || key == "" {
		return fmt.Errorf("hash and key cannot be empty")
	}
	return bcrypt.CompareHashAndPassword([]byte(hashedKey), []byte(key))
}

type DefaultRecordRepository struct{}

func (r *DefaultRecordRepository) FindMatchingApiKeyRecord(
	records []*core.Record,
	apiKey string,
	hasher KeyHasher,
) (*core.Record, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}

	for _, record := range records {
		if record == nil {
			continue
		}

		storedHash := record.GetString("key")
		if storedHash == "" {
			continue
		}

		if err := hasher.CompareHashAndKey(storedHash, apiKey); err == nil {
			return record, nil
		}
	}

	return nil, fmt.Errorf("no matching API key record found")
}

func NewApiKeyService(app App) *ApiKeyService {
	return &ApiKeyService{
		app:              app,
		keyGenerator:     &CryptoKeyGenerator{},
		keyHasher:        NewBcryptKeyHasher(),
		recordRepository: &DefaultRecordRepository{},
	}
}

func NewApiKeyServiceWithDependencies(
	app App,
	keyGen KeyGenerator,
	hasher KeyHasher,
	repo RecordRepository,
) *ApiKeyService {
	return &ApiKeyService{
		app:              app,
		keyGenerator:     keyGen,
		keyHasher:        hasher,
		recordRepository: repo,
	}
}

func (s *ApiKeyService) GenerateApiKey(userId, name string) (string, error) {
	if userId == "" {
		return "", apierror.New(
			http.StatusBadRequest,
			"request.validation",
			"user_id_required",
			"user ID is required",
		)
	}

	if name == "" {
		return "", apierror.New(
			http.StatusBadRequest,
			"request.validation",
			"name_required",
			"name is required",
		)
	}

	apiKeyBytes, err := s.keyGenerator.GenerateKeyBytes()
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_generate_api_key",
			err.Error(),
		)
	}

	apiKey := s.keyGenerator.EncodeKey(apiKeyBytes)
	if apiKey == "" {
		return "", apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_encode_api_key",
			"failed to encode API key",
		)
	}

	hashedKey, err := s.keyHasher.HashKey(apiKey)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_hash_api_key",
			err.Error(),
		)
	}

	apiKeysCollection, err := s.app.FindCollectionByNameOrId("api_keys")
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_find_api_keys_collection",
			err.Error(),
		)
	}

	record := core.NewRecord(apiKeysCollection)
	record.Set("user", userId)
	record.Set("key", hashedKey)
	record.Set("name", name)

	if err := s.app.Save(record); err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_create_api_key_record",
			err.Error(),
		)
	}

	return apiKey, nil
}

func (s *ApiKeyService) AuthenticateApiKey(apiKey string) (*core.Record, error) {
	if apiKey == "" {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"request.validation",
			"api_key_required",
			"API key is required for authentication",
		)
	}

	records, err := s.app.FindRecordsByFilter("api_keys", "", "", 0, 0)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_find_api_key_records",
			err.Error(),
		)
	}

	matchedRecord, err := s.recordRepository.FindMatchingApiKeyRecord(records, apiKey, s.keyHasher)
	if err != nil {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"request.validation",
			"invalid_api_key",
			"Invalid API key provided",
		)
	}

	userId := matchedRecord.GetString("user")
	if userId == "" {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"request.validation",
			"user_not_found",
			"User associated with the API key not found",
		)
	}

	authRecord, err := s.app.FindRecordById("users", userId)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"request.internal_error",
			"failed_to_find_user",
			err.Error(),
		)
	}

	if authRecord == nil {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"request.validation",
			"user_not_found",
			"User associated with the API key not found",
		)
	}

	return authRecord, nil
}
