// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package walletroles

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// The request fields a fixture's behaviour class can be built from: what the
// wallet can show for a verifier request. A request without a scheme asks
// for Capture's default, OpenID4VPRequestScheme.
const (
	// ClassFieldFormats: the formats of dcql_query.credentials, sorted, with counts.
	ClassFieldFormats = "dcql_query.credentials.format"
	// ClassFieldCredentialSets: whether dcql_query has credential_sets.
	ClassFieldCredentialSets = "dcql_query.credential_sets"
	// ClassFieldScheme: the request's URL scheme.
	ClassFieldScheme = "scheme"
	// ClassFieldResponseStatus: the HTTP status the verifier replies with after Share.
	ClassFieldResponseStatus = "response_scenario.status"
	// ClassFieldResponseContentType: the media type of that reply.
	ClassFieldResponseContentType = "response_scenario.content_type"
	// ClassFieldRedirectURI: whether the verifier sends the wallet a redirect_uri.
	ClassFieldRedirectURI = "redirect_uri"
)

// ClassFields lists the known class fields, in the order a class label
// names them.
var ClassFields = []string{
	ClassFieldFormats,
	ClassFieldCredentialSets,
	ClassFieldScheme,
	ClassFieldResponseStatus,
	ClassFieldResponseContentType,
	ClassFieldRedirectURI,
}

// anyRequestClass is the class of a request none of the class fields tells
// apart.
const anyRequestClass = "any request"

// Class returns the behaviour class of a fixture request body: a short
// label built only from the contract's authoring_class_fields, never empty.
// The DCQL query is presentation_request.dcql_query when present, else the
// top-level dcql_query; the scheme is the top-level one, else the
// presentation request's, else OpenID4VPRequestScheme.
func (c *Contract) Class(body any) string {
	request, _ := body.(map[string]any)
	presentation, _ := request["presentation_request"].(map[string]any)
	query, ok := presentation["dcql_query"].(map[string]any)
	if !ok {
		query, _ = request["dcql_query"].(map[string]any)
	}
	listed := func(field string) bool { return slices.Contains(c.AuthoringClassFields, field) }
	parts := []string{}
	if listed(ClassFieldFormats) {
		parts = append(parts, credentialFormats(query))
	}
	if listed(ClassFieldCredentialSets) && query["credential_sets"] != nil {
		parts = append(parts, "with credential_sets")
	}
	if listed(ClassFieldScheme) {
		parts = append(parts, requestScheme(request, presentation))
	}
	response, _ := request["response_scenario"].(map[string]any)
	reply := []string{}
	if status := response["status"]; status != nil && listed(ClassFieldResponseStatus) {
		reply = append(reply, fmt.Sprint(status))
	}
	if kind, _ := response["content_type"].(string); kind != "" &&
		listed(ClassFieldResponseContentType) {
		reply = append(reply, kind)
	}
	if len(reply) > 0 {
		parts = append(parts, "verifier replies "+strings.Join(reply, " "))
	}
	if uri := request["redirect_uri"]; uri != nil && uri != "" && listed(ClassFieldRedirectURI) {
		parts = append(parts, "with redirect_uri")
	}
	if len(parts) == 0 {
		return anyRequestClass
	}
	return strings.Join(parts, ", ")
}

// credentialFormats names the formats of a DCQL query's credentials, sorted,
// each with its count when above one: "2x dc+sd-jwt + mso_mdoc".
func credentialFormats(query map[string]any) string {
	credentials, _ := query["credentials"].([]any)
	counts := map[string]int{}
	for _, credential := range credentials {
		entry, _ := credential.(map[string]any)
		if format, _ := entry["format"].(string); format != "" {
			counts[format]++
		}
	}
	if len(counts) == 0 {
		return "no dcql credentials"
	}
	formats := make([]string, 0, len(counts))
	for format := range counts {
		formats = append(formats, format)
	}
	sort.Strings(formats)
	for i, format := range formats {
		if counts[format] > 1 {
			formats[i] = fmt.Sprintf("%dx %s", counts[format], format)
		}
	}
	return strings.Join(formats, " + ")
}

// requestScheme is the scheme a request asks for, Capture's default when it
// names none.
func requestScheme(request, presentation map[string]any) string {
	if scheme, _ := request["scheme"].(string); scheme != "" {
		return scheme
	}
	if scheme, _ := presentation["scheme"].(string); scheme != "" {
		return scheme
	}
	return OpenID4VPRequestScheme
}

// AuthoringFixtures returns one representative fixture per behaviour class:
// of each class the fixture the action author replays most usefully, that
// is one with tests (a fixture without tests represents its class only when
// no fixture of the class has tests), then standalone, then one whose setup
// is complete, then the one that feeds the most tests, then by ID. Classes
// come standalone first, then complete, then by their representative's test
// count, most first, then by label.
func AuthoringFixtures(fixtures []Fixture) []Fixture {
	representatives := map[string]Fixture{}
	for _, fixture := range fixtures {
		current, seen := representatives[fixture.Class]
		if !seen || representsBefore(fixture, current) {
			representatives[fixture.Class] = fixture
		}
	}
	set := make([]Fixture, 0, len(representatives))
	for _, fixture := range representatives {
		set = append(set, fixture)
	}
	sort.Slice(set, func(i, j int) bool {
		a, b := set[i], set[j]
		if startRank(a) != startRank(b) {
			return startRank(a) < startRank(b)
		}
		if len(a.TestIDs) != len(b.TestIDs) {
			return len(a.TestIDs) > len(b.TestIDs)
		}
		return a.Class < b.Class
	})
	return set
}

// representsBefore reports whether a represents its class better than b.
func representsBefore(a, b Fixture) bool {
	if tested := len(a.TestIDs) > 0; tested != (len(b.TestIDs) > 0) {
		return tested
	}
	if startRank(a) != startRank(b) {
		return startRank(a) < startRank(b)
	}
	if len(a.TestIDs) != len(b.TestIDs) {
		return len(a.TestIDs) > len(b.TestIDs)
	}
	return a.ID < b.ID
}

// startRank orders fixtures by how fully a replay rebuilds their start
// state: standalone, then complete setup, then the others.
func startRank(fixture Fixture) int {
	switch {
	case fixture.Standalone:
		return 0
	case fixture.Complete:
		return 1
	default:
		return 2
	}
}

// AuthoringSet returns the fixtures of a role the action author replays: for
// the injected PID issuance every fixture in Fixtures order (the runner picks
// the wallet's PID issuer among them), for any other role AuthoringFixtures
// then the role's pinned authoring_fixtures that are not representatives,
// in Fixtures order. A pinned ID that is not a fixture of the role is an error.
func (c *Contract) AuthoringSet(suite, roleName string) ([]Fixture, error) {
	fixtures, err := c.Fixtures(c.root, suite, roleName)
	if err != nil {
		return nil, err
	}
	role, _ := c.Role(roleName)
	if role.Injected == InjectedIssuance {
		return fixtures, nil
	}
	set := AuthoringFixtures(fixtures)
	for _, fixture := range fixtures {
		if slices.Contains(role.AuthoringFixtures, fixture.ID) &&
			!slices.ContainsFunc(set, func(f Fixture) bool { return f.ID == fixture.ID }) {
			set = append(set, fixture)
		}
	}
	for _, id := range role.AuthoringFixtures {
		if !slices.ContainsFunc(fixtures, func(f Fixture) bool { return f.ID == id }) {
			return nil, fmt.Errorf(
				"role %q: authoring fixture %q is not one of its fixtures",
				roleName,
				id,
			)
		}
	}
	return set, nil
}
