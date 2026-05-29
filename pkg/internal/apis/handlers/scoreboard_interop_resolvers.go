// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
)

type interopRelatedSpec struct {
	Collection string
	Field      string
}

type interopEntityResolver interface {
	Collection() string
	RelatedCollections() []interopRelatedSpec
	SupportsVersionLabels() bool
	Entity(app core.App, record *core.Record, related interopRelatedRecords) (InteropMatrixEntity, error)
}

var interopEntityResolvers = map[string]interopEntityResolver{
	"wallets":                 &simpleInteropEntityResolver{collection: "wallets", avatarFields: []string{"avatar", "logo", "logo_url"}, supportsVersionLabels: true},
	"credential_issuers":      &simpleInteropEntityResolver{collection: "credential_issuers", avatarFields: []string{"avatar", "logo", "logo_url"}},
	"verifiers":               &simpleInteropEntityResolver{collection: "verifiers", avatarFields: []string{"avatar", "logo"}},
	"credentials":             &credentialsInteropEntityResolver{},
	"use_cases_verifications": &useCasesVerificationsInteropEntityResolver{},
}

func getInteropEntityResolver(collection string) (interopEntityResolver, error) {
	resolver, ok := interopEntityResolvers[collection]
	if !ok {
		return nil, fmt.Errorf("no interop entity resolver for collection %s", collection)
	}
	return resolver, nil
}

func interopEntityPath(app core.App, record *core.Record, collection string) (string, error) {
	tpl, ok := canonify.CanonifyPaths[collection]
	if !ok {
		return "", fmt.Errorf("no canonify path for collection %s", collection)
	}
	return canonify.BuildPath(app, record, tpl, "")
}

type simpleInteropEntityResolver struct {
	collection            string
	avatarFields          []string
	supportsVersionLabels bool
}

func (r *simpleInteropEntityResolver) Collection() string { return r.collection }

func (r *simpleInteropEntityResolver) RelatedCollections() []interopRelatedSpec { return nil }

func (r *simpleInteropEntityResolver) SupportsVersionLabels() bool { return r.supportsVersionLabels }

func (r *simpleInteropEntityResolver) Entity(
	app core.App,
	record *core.Record,
	_ interopRelatedRecords,
) (InteropMatrixEntity, error) {
	path, err := interopEntityPath(app, record, r.collection)
	if err != nil {
		return InteropMatrixEntity{}, err
	}

	avatarValues := make([]string, len(r.avatarFields))
	for i, field := range r.avatarFields {
		avatarValues[i] = record.GetString(field)
	}

	return InteropMatrixEntity{
		ID:        record.Id,
		Name:      record.GetString("name"),
		Path:      path,
		AvatarURL: firstNonEmptyStringPtr(avatarValues...),
	}, nil
}

type credentialsInteropEntityResolver struct{}

func (r *credentialsInteropEntityResolver) Collection() string { return "credentials" }

func (r *credentialsInteropEntityResolver) RelatedCollections() []interopRelatedSpec {
	return []interopRelatedSpec{{Collection: "credential_issuers", Field: "credential_issuer"}}
}

func (r *credentialsInteropEntityResolver) SupportsVersionLabels() bool { return false }

func (r *credentialsInteropEntityResolver) Entity(
	app core.App,
	record *core.Record,
	related interopRelatedRecords,
) (InteropMatrixEntity, error) {
	path, err := interopEntityPath(app, record, r.Collection())
	if err != nil {
		return InteropMatrixEntity{}, err
	}

	var issuerName *string
	var issuerAvatarURL *string
	issuerID := strings.TrimSpace(record.GetString("credential_issuer"))
	if issuerID != "" {
		if issuerRecord := related.record("credential_issuers", issuerID); issuerRecord != nil {
			issuerName = optionalTrimmedStringPtr(issuerRecord.GetString("name"))
			issuerAvatarURL = firstNonEmptyStringPtr(
				issuerRecord.GetString("avatar"),
				issuerRecord.GetString("logo_url"),
			)
		}
	}

	return buildEnrichedEntityMetadata(
		record.Id,
		record.GetString("name"),
		path,
		firstNonEmptyStringPtr(record.GetString("avatar"), record.GetString("logo_url")),
		issuerName,
		issuerAvatarURL,
	), nil
}

type useCasesVerificationsInteropEntityResolver struct{}

func (r *useCasesVerificationsInteropEntityResolver) Collection() string {
	return "use_cases_verifications"
}

func (r *useCasesVerificationsInteropEntityResolver) RelatedCollections() []interopRelatedSpec {
	return []interopRelatedSpec{{Collection: "verifiers", Field: "verifier"}}
}

func (r *useCasesVerificationsInteropEntityResolver) SupportsVersionLabels() bool { return false }

func (r *useCasesVerificationsInteropEntityResolver) Entity(
	app core.App,
	record *core.Record,
	related interopRelatedRecords,
) (InteropMatrixEntity, error) {
	path, err := interopEntityPath(app, record, r.Collection())
	if err != nil {
		return InteropMatrixEntity{}, err
	}

	var verifierName *string
	var verifierLogoURL *string
	verifierID := strings.TrimSpace(record.GetString("verifier"))
	if verifierID != "" {
		if verifierRecord := related.record("verifiers", verifierID); verifierRecord != nil {
			verifierName = optionalTrimmedStringPtr(verifierRecord.GetString("name"))
			verifierLogoURL = firstNonEmptyStringPtr(
				verifierRecord.GetString("avatar"),
				verifierRecord.GetString("logo"),
			)
		}
	}

	return buildEnrichedEntityMetadata(
		record.Id,
		record.GetString("name"),
		path,
		firstNonEmptyStringPtr(record.GetString("avatar"), record.GetString("logo")),
		verifierName,
		verifierLogoURL,
	), nil
}
