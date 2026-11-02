// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Kinds of a ReferenceResolution.
const (
	ReferenceCredimiRecord       = "credimi_record"
	ReferenceCredentialOffer     = "credential_offer_deeplink"
	ReferencePresentation        = "presentation_deeplink"
	ReferenceUnknown             = "unknown"
	credentialOfferScheme        = "openid-credential-offer"
	referenceNoOfferMatch        = "no Credimi credential matches this offer; a deeplink works once, so replays need a source that creates a new one: a Credimi credential, or an HTTP call that creates an issuer session"
	referencePresentationNote    = "Credimi records do not identify verifiers by client_id; replays need a Credimi use case verification, or an HTTP call that creates a verifier session"
	referenceUnknownNote         = "not a Credimi URL, identifier or deeplink"
	referenceRecordNotFoundNote  = "%s not found in your organization's or published records"
	hubPathPrefix                = "/hub/"
	marketplacePathPrefix        = "/marketplace/"
	referenceCredentialsSegment  = "credentials"
	referenceVerificationSegment = "use_cases_verifications"
)

// SourceBodyLimit bounds the bodies Credimi reads from the http sources and
// credential offer URIs the agent names.
const SourceBodyLimit = 1 << 20

var (
	canonifiedReference = regexp.MustCompile(`^[^/\s]+/[^/\s]+/[^/\s]+$`)
	// presentationSchemes are the deeplink schemes of presentation requests.
	presentationSchemes = map[string]bool{
		"haip-vp":        true,
		"openid4vp":      true,
		"eudi-openid4vp": true,
		"mdoc-openid4vp": true,
	}
	// referenceSourceKinds maps the record collections a reference can name
	// to their parameter source kind.
	referenceSourceKinds = map[string]string{
		referenceCredentialsSegment:  SourceCredentialOffer,
		referenceVerificationSegment: SourceUseCaseVerification,
	}
)

// ReferenceResolution classifies a reference the user gave the agent and
// the parameter sources it maps to.
type ReferenceResolution struct {
	Kind    string            `json:"kind"`
	Sources []ReferenceSource `json:"sources"`
	Decoded map[string]string `json:"decoded,omitempty"`
	Note    string            `json:"note,omitempty"`
}

// ReferenceSource is a parameter source a reference maps to.
type ReferenceSource struct {
	Label  string     `json:"label"`
	Source SourceSpec `json:"source"`
}

// ResolveReference turns a Credimi URL, canonified path or deeplink into the
// parameter sources the organization can use. client fetches
// credential_offer_uri; a raw deeplink is never opened.
func ResolveReference(
	app core.App,
	orgID, text string,
	client *http.Client,
) (ReferenceResolution, error) {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)
	switch {
	case strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://"):
		return resolveHubURL(app, orgID, text)
	case canonifiedReference.MatchString(text):
		return resolveRecordPath(
			app,
			orgID,
			text,
			referenceCredentialsSegment,
			referenceVerificationSegment,
		)
	}
	parsed, err := url.Parse(text)
	if err != nil {
		parsed = &url.URL{}
	}
	scheme := strings.ToLower(parsed.Scheme)
	switch {
	case scheme == credentialOfferScheme:
		return resolveCredentialOffer(app, orgID, parsed, client)
	case presentationSchemes[scheme]:
		decoded := map[string]string{"client_id": parsed.Query().Get("client_id")}
		if requestURI := parsed.Query().Get("request_uri"); requestURI != "" {
			decoded["request_uri"] = requestURI
		}
		return ReferenceResolution{
			Kind:    ReferencePresentation,
			Sources: []ReferenceSource{},
			Decoded: decoded,
			Note:    referencePresentationNote,
		}, nil
	default:
		return unknownReference(), nil
	}
}

func unknownReference() ReferenceResolution {
	return ReferenceResolution{
		Kind:    ReferenceUnknown,
		Sources: []ReferenceSource{},
		Note:    referenceUnknownNote,
	}
}

// resolveHubURL resolves a Credimi hub (or marketplace) URL of a credential
// or use case verification.
func resolveHubURL(app core.App, orgID, text string) (ReferenceResolution, error) {
	parsed, err := url.Parse(text)
	if err != nil {
		parsed = &url.URL{}
	}
	path := parsed.Path
	if rest, ok := strings.CutPrefix(path, marketplacePathPrefix); ok {
		path = hubPathPrefix + rest
	}
	for _, collection := range []string{referenceCredentialsSegment, referenceVerificationSegment} {
		marker := hubPathPrefix + collection + "/"
		if _, rest, ok := strings.Cut(path, marker); ok {
			return resolveRecordPath(app, orgID, strings.TrimSuffix(rest, "/"), collection)
		}
	}
	return unknownReference(), nil
}

// resolveRecordPath returns a source for each collection whose record at
// path the organization can use.
func resolveRecordPath(
	app core.App,
	orgID, path string,
	collections ...string,
) (ReferenceResolution, error) {
	resolution := ReferenceResolution{Kind: ReferenceCredimiRecord, Sources: []ReferenceSource{}}
	for _, collection := range collections {
		record, err := CheckRecordAccess(app, collection, path, orgID)
		if errors.Is(err, ErrInvalid) {
			continue
		}
		if err != nil {
			return ReferenceResolution{}, err
		}
		source, err := recordSource(app, collection, record)
		if err != nil {
			return ReferenceResolution{}, err
		}
		resolution.Sources = append(resolution.Sources, source)
	}
	if len(resolution.Sources) == 0 {
		resolution.Note = fmt.Sprintf(referenceRecordNotFoundNote, path)
	}
	return resolution, nil
}

func recordSource(app core.App, collection string, record *core.Record) (ReferenceSource, error) {
	path, err := canonify.BuildPath(
		app,
		record,
		canonify.CanonifyPaths[collection],
		record.GetString("canonified_name"),
	)
	if err != nil {
		return ReferenceSource{}, fmt.Errorf("build %s path: %w", collection, err)
	}
	return ReferenceSource{
		Label:  record.GetString("name"),
		Source: SourceSpec{Kind: referenceSourceKinds[collection], Identifier: path},
	}, nil
}

// credentialOffer is the part of an OpenID4VCI credential offer that
// identifies its credentials.
type credentialOffer struct {
	CredentialIssuer           string   `json:"credential_issuer"`
	CredentialConfigurationIDs []string `json:"credential_configuration_ids"`
}

// resolveCredentialOffer decodes a credential offer deeplink and matches it
// against the Credimi credentials the organization can use.
func resolveCredentialOffer(
	app core.App,
	orgID string,
	deeplink *url.URL,
	client *http.Client,
) (ReferenceResolution, error) {
	resolution := ReferenceResolution{Kind: ReferenceCredentialOffer, Sources: []ReferenceSource{}}
	offer, problem := readCredentialOffer(deeplink, client)
	if problem != "" {
		resolution.Note = "cannot read the offer: " + problem
		return resolution, nil
	}
	resolution.Decoded = map[string]string{
		"credential_issuer":            offer.CredentialIssuer,
		"credential_configuration_ids": strings.Join(offer.CredentialConfigurationIDs, ","),
	}
	sources, err := matchOfferCredentials(app, orgID, offer)
	if err != nil {
		return ReferenceResolution{}, err
	}
	resolution.Sources = sources
	if len(sources) == 0 {
		resolution.Note = referenceNoOfferMatch
	}
	return resolution, nil
}

// readCredentialOffer decodes the offer of a deeplink, or says why it cannot.
func readCredentialOffer(deeplink *url.URL, client *http.Client) (credentialOffer, string) {
	var offer credentialOffer
	query := deeplink.Query()
	raw := []byte(query.Get("credential_offer"))
	if len(raw) == 0 {
		offerURI := query.Get("credential_offer_uri")
		if offerURI == "" {
			return offer, "the deeplink has no credential_offer or credential_offer_uri"
		}
		fetched, err := fetchCredentialOffer(client, offerURI)
		if err != nil {
			return offer, err.Error()
		}
		raw = fetched
	}
	if err := json.Unmarshal(raw, &offer); err != nil {
		return offer, "the offer is not JSON: " + err.Error()
	}
	if offer.CredentialIssuer == "" || len(offer.CredentialConfigurationIDs) == 0 {
		return offer, "the offer has no credential_issuer or credential_configuration_ids"
	}
	return offer, ""
}

func fetchCredentialOffer(client *http.Client, offerURI string) ([]byte, error) {
	// client bounds the request with its timeout.
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		offerURI,
		nil,
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("credential_offer_uri answered HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, SourceBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if len(body) > SourceBodyLimit {
		return nil, fmt.Errorf("the offer exceeds %d bytes", SourceBodyLimit)
	}
	return body, nil
}

// matchOfferCredentials returns the usable credentials named by one of the
// offer's configuration IDs whose issuer URL is the offer's issuer.
func matchOfferCredentials(
	app core.App,
	orgID string,
	offer credentialOffer,
) ([]ReferenceSource, error) {
	ids := make([]any, 0, len(offer.CredentialConfigurationIDs))
	for _, id := range offer.CredentialConfigurationIDs {
		ids = append(ids, id)
	}
	records, err := app.FindAllRecords(referenceCredentialsSegment, dbx.In("name", ids...))
	if err != nil {
		return nil, fmt.Errorf("find offered credentials: %w", err)
	}
	issuerURL := strings.TrimSuffix(offer.CredentialIssuer, "/")
	sources := []ReferenceSource{}
	for _, record := range records {
		issuer, err := app.FindRecordById(
			"credential_issuers",
			record.GetString("credential_issuer"),
		)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("find credential issuer: %w", err)
		}
		if strings.TrimSuffix(issuer.GetString("url"), "/") != issuerURL {
			continue
		}
		source, err := recordSource(app, referenceCredentialsSegment, record)
		if err != nil {
			return nil, err
		}
		_, err = CheckRecordAccess(
			app,
			referenceCredentialsSegment,
			source.Source.Identifier,
			orgID,
		)
		if errors.Is(err, ErrInvalid) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}
