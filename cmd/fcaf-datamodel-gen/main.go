// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command fcaf-datamodel-gen translates the reviewed DataModel mapping below
// into runtime FCAF YAML definitions.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"gopkg.in/yaml.v3"
)

const (
	defaultSource = "../eudi-doc-functional-conformance-assessment/docs/fcaf/suts/wallet_solution/relying_party/DataModel"
	defaultOutput = "config_templates/fcaf/wallet_solution/relying_party/tests"
)

var variantPattern = regexp.MustCompile(`_(00[1-9]|010)$`)

type assertionSpec struct {
	id        string
	validator string
	params    map[string]any
}

type familySpec struct {
	marker     string
	claim      string
	assertions map[string]assertionSpec
}

func main() {
	source := flag.String("source", defaultSource, "DataModel Markdown source directory")
	output := flag.String("output", defaultOutput, "FCAF test YAML output directory")
	format := flag.String("format", "all", "format to generate: all, sdjwt, or mdoc")
	flag.Parse()

	switch *format {
	case "all":
		if err := generateSDJWT(*source, *output); err != nil {
			fatalf("%v", err)
		}
		if err := generateMDoc(*source, *output); err != nil {
			fatalf("%v", err)
		}
	case "sdjwt":
		if err := generateSDJWT(*source, *output); err != nil {
			fatalf("%v", err)
		}
	case "mdoc":
		if err := generateMDoc(*source, *output); err != nil {
			fatalf("%v", err)
		}
	default:
		fatalf("unsupported format %q", *format)
	}
}

func generateSDJWT(source string, output string) error {
	specs := sdjwtFamilySpecs()
	files, err := filepath.Glob(filepath.Join(source, "*", "*IETF-sd-jwt-vc*.md"))
	if err != nil {
		return fmt.Errorf("list SD-JWT DataModel sources: %w", err)
	}
	sort.Strings(files)
	if len(files) != 66 {
		return fmt.Errorf("expected 66 SD-JWT DataModel sources, found %d", len(files))
	}

	for _, path := range files {
		id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		spec, variant, err := matchSpec(id, specs)
		if err != nil {
			return err
		}
		def, err := buildDefinition(source, path, id, variant, spec)
		if err != nil {
			return err
		}
		raw, err := yaml.Marshal(def)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", id, err)
		}
		if err := os.WriteFile(filepath.Join(output, id+".yaml"), raw, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", id, err)
		}
	}
	return nil
}

func generateMDoc(source string, output string) error {
	specs := mdocFamilySpecs()
	files, err := filepath.Glob(filepath.Join(source, "*", "*ISO-mdoc*.md"))
	if err != nil {
		return fmt.Errorf("list mdoc DataModel sources: %w", err)
	}
	sort.Strings(files)
	if len(files) != 72 {
		return fmt.Errorf("expected 72 mdoc DataModel sources, found %d", len(files))
	}

	for _, path := range files {
		id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		spec, variant, err := matchSpec(id, specs)
		if err != nil {
			return err
		}
		def, err := buildMDocDefinition(source, path, id, variant, spec)
		if err != nil {
			return err
		}
		raw, err := yaml.Marshal(def)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", id, err)
		}
		if err := os.WriteFile(filepath.Join(output, id+".yaml"), raw, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", id, err)
		}
	}
	return nil
}

func buildDefinition(
	sourceRoot string,
	path string,
	id string,
	variant string,
	spec familySpec,
) (dsl.TestDefinition, error) {
	sections, err := readMarkdownSections(path)
	if err != nil {
		return dsl.TestDefinition{}, err
	}
	assertion, ok := spec.assertions[variant]
	if !ok {
		return dsl.TestDefinition{}, fmt.Errorf(
			"%s has no reviewed assertion mapping for variant %s",
			id,
			variant,
		)
	}
	relative, err := filepath.Rel(sourceRoot, path)
	if err != nil {
		return dsl.TestDefinition{}, fmt.Errorf("relative source path: %w", err)
	}
	references := make([]dsl.NormativeReference, 0, len(sections["References"]))
	sourceURL := "https://conformance.eudi.dev/latest/fcaf/suts/wallet_solution/relying_party/ws_rp/#" +
		strings.ToLower(
			id,
		)
	for _, reference := range sections["References"] {
		reference = cleanMarkdownLine(reference)
		if reference == "" {
			continue
		}
		references = append(references, dsl.NormativeReference{
			Title: reference,
			URL:   sourceURL,
		})
	}
	if len(references) == 0 {
		return dsl.TestDefinition{}, fmt.Errorf("%s has no normative references", id)
	}

	preconditions := []dsl.PreconditionRef{
		{Ref: "pipeline.pid.presentation.sdjwt.all-ics-claims"},
		{Ref: "assertion.pid.presentation.sdjwt.vct-pid"},
		{Ref: "assertion.pid.presentation.sdjwt.required-mandatory-claims-presented"},
	}
	if variant != "001" {
		preconditions = append(preconditions, dsl.PreconditionRef{
			Ref: "test." + variantPattern.ReplaceAllString(id, "_001"),
		})
	}

	title := strings.Join(sections["Objective"], " ")
	title = strings.TrimSpace(strings.ReplaceAll(title, "\r", ""))
	if title == "" {
		title = id
	}
	section := sectionName(filepath.Dir(relative))
	return dsl.TestDefinition{
		ID:    id,
		Title: title,
		Source: dsl.Source{
			Path: filepath.ToSlash(filepath.Join("DataModel", relative)),
		},
		Suite: dsl.Suite{
			SUT:     "wallet_solution",
			Role:    "relying_party",
			Section: "data_model." + section,
		},
		Applicability: map[string]any{
			"credential_format": "ietf_sd_jwt_vc",
			"document_type":     "pid",
		},
		NormativeReferences: references,
		Preconditions:       preconditions,
		Evidence: map[string]dsl.EvidenceBinding{
			"pid_sdjwt": {
				From: "pipeline.pid.presentation.sdjwt.all-ics-claims.outputs.pid_sdjwt",
			},
		},
		Assertions: []dsl.AssertionDefinition{{
			ID:        assertion.id,
			Validator: assertion.validator,
			Input:     "evidence.pid_sdjwt",
			Params:    assertion.params,
		}},
		Verdict: dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
	}, nil
}

func buildMDocDefinition(
	sourceRoot string,
	path string,
	id string,
	variant string,
	spec familySpec,
) (dsl.TestDefinition, error) {
	sections, err := readMarkdownSections(path)
	if err != nil {
		return dsl.TestDefinition{}, err
	}
	assertion, ok := spec.assertions[variant]
	if !ok {
		return dsl.TestDefinition{}, fmt.Errorf(
			"%s has no reviewed assertion mapping for variant %s",
			id,
			variant,
		)
	}
	relative, err := filepath.Rel(sourceRoot, path)
	if err != nil {
		return dsl.TestDefinition{}, fmt.Errorf("relative source path: %w", err)
	}
	sourceURL := "https://conformance.eudi.dev/latest/fcaf/suts/wallet_solution/relying_party/ws_rp/#" +
		strings.ToLower(
			id,
		)
	references := make([]dsl.NormativeReference, 0, len(sections["References"]))
	for _, reference := range sections["References"] {
		reference = cleanMarkdownLine(reference)
		if reference != "" {
			references = append(
				references,
				dsl.NormativeReference{Title: reference, URL: sourceURL},
			)
		}
	}
	if len(references) == 0 {
		return dsl.TestDefinition{}, fmt.Errorf("%s has no normative references", id)
	}
	preconditions := []dsl.PreconditionRef{
		{Ref: "pipeline.pid.presentation.mdoc.all-ics-elements"},
		{Ref: "assertion.pid.presentation.mdoc.doc-type-pid"},
		{Ref: "assertion.pid.presentation.mdoc.required-mandatory-elements-presented"},
	}
	if variant != "001" {
		preconditions = append(preconditions, dsl.PreconditionRef{
			Ref: "test." + variantPattern.ReplaceAllString(id, "_001"),
		})
	}
	title := strings.TrimSpace(strings.Join(sections["Objective"], " "))
	if title == "" {
		title = id
	}
	return dsl.TestDefinition{
		ID:     id,
		Title:  title,
		Source: dsl.Source{Path: filepath.ToSlash(filepath.Join("DataModel", relative))},
		Suite: dsl.Suite{
			SUT:     "wallet_solution",
			Role:    "relying_party",
			Section: "data_model." + sectionName(filepath.Dir(relative)),
		},
		Applicability: map[string]any{
			"credential_format": "iso_mdoc",
			"document_type":     "pid",
		},
		NormativeReferences: references,
		Preconditions:       preconditions,
		Evidence: map[string]dsl.EvidenceBinding{
			"pid_mdoc": {From: "pipeline.pid.presentation.mdoc.all-ics-elements.outputs.pid_mdoc"},
		},
		Assertions: []dsl.AssertionDefinition{{
			ID:        assertion.id,
			Validator: assertion.validator,
			Input:     "evidence.pid_mdoc",
			Params:    assertion.params,
		}},
		Verdict: dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
	}, nil
}

func sectionName(directory string) string {
	switch filepath.Base(directory) {
	case "AddressData":
		return "address_data"
	case "CredentialMetadata":
		return "credential_metadata"
	case "IdentifyingData":
		return "identifying_data"
	default:
		return strings.ToLower(filepath.Base(directory))
	}
}

func sdjwtFamilySpecs() []familySpec {
	standard := func(marker string, claim string) familySpec {
		return familySpec{
			marker: marker,
			claim:  claim,
			assertions: map[string]assertionSpec{
				"001": claimAssertion(claim, "present", "sdjwt.claim_present", nil),
				"002": claimAssertion(claim, "utf8", "sdjwt.claim_utf8_string", nil),
			},
		}
	}
	semantic := func(marker string, claim string, validator string, params map[string]any) familySpec {
		spec := standard(marker, claim)
		spec.assertions["003"] = claimAssertion(claim, "semantic", validator, params)
		return spec
	}
	date := func(marker string, claim string) familySpec {
		spec := standard(marker, claim)
		spec.assertions["003"] = claimAssertion(claim, "format", "sdjwt.claim_date_format", nil)
		spec.assertions["004"] = claimAssertion(claim, "valid", "sdjwt.claim_valid_date", nil)
		return spec
	}

	specs := []familySpec{
		semantic("AddressData_Emailaddress_", "email", "sdjwt.claim_rfc5322_email", nil),
		semantic(
			"AddressData_Mobilephonenumber_",
			"phone_number",
			"sdjwt.claim_international_phone",
			map[string]any{"min_length": 8},
		),
		standard("AddressData_Residentaddress_", "address.formatted"),
		standard("AddressData_Residentcity_", "address.locality"),
		semantic(
			"AddressData_Residentcountry_",
			"address.country",
			"sdjwt.claim_country_code",
			nil,
		),
		standard("AddressData_Residenthousenumber_", "address.house_number"),
		standard("AddressData_Residentpostalcode_", "address.postal_code"),
		standard("AddressData_Residentstate_", "address.region"),
		standard("AddressData_Residentstreet_", "address.street_address"),
		standard("Credentialmetadata_Documentnumber_", "document_number"),
		date("Credentialmetadata_Expirydate_", "date_of_expiry"),
		date("Credentialmetadata_Issuancedate_", "date_of_issuance"),
		standard("Credentialmetadata_Issuingauthority_", "issuing_authority"),
		semantic(
			"Credentialmetadata_Issuingcountry_",
			"issuing_country",
			"sdjwt.claim_country_code",
			nil,
		),
		semantic(
			"Credentialmetadata_Issuingjurisdiction_",
			"issuing_jurisdiction",
			"sdjwt.claim_country_subdivision",
			map[string]any{
				"country_claim": "issuing_country",
			},
		),
		date("IdentifyingData_Birthdate_", "birthdate"),
		standard("IdentifyingData_Familyname_", "family_name"),
		standard("IdentifyingData_Familynamebirth_", "birth_family_name"),
		standard("IdentifyingData_Givenname_", "given_name"),
		standard("IdentifyingData_Givennamebirth_", "birth_given_name"),
		standard("IdentifyingData_Personaladministrativenumber_", "personal_administrative_number"),
	}
	specs = append(specs,
		familySpec{
			marker: "Credentialmetadata_Domestic_",
			assertions: map[string]assertionSpec{
				"001": {id: "domestic_claims_present", validator: "sdjwt.domestic_namespace"},
				"002": {id: "domestic_namespace_valid", validator: "sdjwt.domestic_namespace"},
			},
		},
		familySpec{
			marker: "IdentifyingData_Birthplace_",
			claim:  "place_of_birth",
			assertions: map[string]assertionSpec{
				"001": claimAssertion("place_of_birth", "present", "sdjwt.claim_present", nil),
				"002": claimAssertion("place_of_birth", "object", "sdjwt.claim_object", nil),
			},
		},
		familySpec{
			marker: "IdentifyingData_Nationality_",
			claim:  "nationalities",
			assertions: map[string]assertionSpec{
				"001": claimAssertion("nationalities", "present", "sdjwt.claim_present", nil),
				"002": claimAssertion(
					"nationalities",
					"strings",
					"sdjwt.claim_string_array",
					map[string]any{"min_items": 1},
				),
				"003": claimAssertion(
					"nationalities",
					"countries",
					"sdjwt.claim_country_code_array",
					map[string]any{"min_items": 1},
				),
			},
		},
		familySpec{
			marker: "IdentifyingData_Portrait_",
			claim:  "picture",
			assertions: map[string]assertionSpec{
				"001": claimAssertion("picture", "present", "sdjwt.claim_present", nil),
				"002": claimAssertion(
					"picture",
					"string",
					"sdjwt.claim_type",
					map[string]any{"type": "string"},
				),
				"003": claimAssertion("picture", "jpeg", "sdjwt.claim_jpeg_data_url", nil),
			},
		},
		familySpec{
			marker: "IdentifyingData_Sex_",
			claim:  "sex",
			assertions: map[string]assertionSpec{
				"001": claimAssertion("sex", "present", "sdjwt.claim_present", nil),
				"002": claimAssertion(
					"sex",
					"number",
					"sdjwt.claim_type",
					map[string]any{"type": "number"},
				),
				"003": claimAssertion(
					"sex",
					"allowed",
					"sdjwt.claim_integer_allowed",
					map[string]any{
						"allowed": []int{0, 1, 2, 3, 4, 5, 6, 9},
					},
				),
			},
		},
	)
	return specs
}

func mdocFamilySpecs() []familySpec {
	standard := func(marker string, element string) familySpec {
		return familySpec{
			marker: marker,
			claim:  element,
			assertions: map[string]assertionSpec{
				"001": elementAssertion(element, "present", "mdoc.namespace_element_present", nil),
				"002": elementAssertion(element, "utf8", "mdoc.element_utf8_string", nil),
			},
		}
	}
	semantic := func(marker string, element string, validator string, params map[string]any) familySpec {
		spec := standard(marker, element)
		spec.assertions["003"] = elementAssertion(element, "semantic", validator, params)
		return spec
	}
	date := func(marker string, element string, allowedTags []int) familySpec {
		return familySpec{
			marker: marker,
			claim:  element,
			assertions: map[string]assertionSpec{
				"001": elementAssertion(element, "present", "mdoc.namespace_element_present", nil),
				"002": elementAssertion(
					element,
					"encoding",
					"mdoc.element_date_encoding",
					map[string]any{
						"allowed_tags": allowedTags,
					},
				),
				"003": elementAssertion(element, "format", "mdoc.element_date_format", nil),
				"004": elementAssertion(element, "valid", "mdoc.element_valid_date", nil),
			},
		}
	}

	specs := []familySpec{
		semantic("AddressData_Emailaddress_", "email_address", "mdoc.element_rfc5322_email", nil),
		semantic(
			"AddressData_Mobilephonenumber_",
			"mobile_phone_number",
			"mdoc.element_international_phone",
			map[string]any{"min_length": 8},
		),
		standard("AddressData_Residentaddress_", "resident_address"),
		standard("AddressData_Residentcity_", "resident_city"),
		semantic(
			"AddressData_Residentcountry_",
			"resident_country",
			"mdoc.element_country_code",
			nil,
		),
		standard("AddressData_Residenthousenumber_", "resident_house_number"),
		standard("AddressData_Residentpostalcode_", "resident_postal_code"),
		standard("AddressData_Residentstate_", "resident_state"),
		standard("AddressData_Residentstreet_", "resident_street"),
		standard("Credentialmetadata_Documentnumber_", "document_number"),
		date("Credentialmetadata_Expirydate_", "expiry_date", []int{0, 1004}),
		date("Credentialmetadata_Issuancedate_", "issuance_date", []int{0, 1004}),
		standard("Credentialmetadata_Issuingauthority_", "issuing_authority"),
		semantic(
			"Credentialmetadata_Issuingcountry_",
			"issuing_country",
			"mdoc.element_country_code",
			nil,
		),
		semantic(
			"Credentialmetadata_Issuingjurisdiction_",
			"issuing_jurisdiction",
			"mdoc.element_country_subdivision",
			map[string]any{
				"country_element": "issuing_country",
			},
		),
		date("IdentifyingData_Birthdate_", "birth_date", []int{1004}),
		standard("IdentifyingData_Familyname_", "family_name"),
		standard("IdentifyingData_Familynamebirth_", "family_name_birth"),
		standard("IdentifyingData_Givenname_", "given_name"),
		standard("IdentifyingData_Givennamebirth_", "given_name_birth"),
		standard("IdentifyingData_Personaladministrativenumber_", "personal_administrative_number"),
	}
	specs = append(specs,
		familySpec{
			marker: "Credentialmetadata_Domestic_",
			assertions: map[string]assertionSpec{
				"001": {id: "domestic_elements_present", validator: "mdoc.domestic_namespace"},
				"002": {id: "domestic_namespace_valid", validator: "mdoc.domestic_namespace"},
			},
		},
		familySpec{
			marker: "IdentifyingData_Birthplace_",
			claim:  "place_of_birth",
			assertions: map[string]assertionSpec{
				"001": elementAssertion(
					"place_of_birth",
					"present",
					"mdoc.namespace_element_present",
					nil,
				),
				"002": elementAssertion(
					"place_of_birth",
					"map",
					"mdoc.element_cbor_type",
					map[string]any{"major_type": 5},
				),
				"003": elementAssertion(
					"place_of_birth",
					"shape",
					"mdoc.element_map_shape",
					map[string]any{
						"allowed_keys": []string{
							"country",
							"region",
							"locality",
						},
						"min_properties": 1,
						"max_properties": 3,
					},
				),
				"004": elementAssertion(
					"place_of_birth",
					"text_values",
					"mdoc.element_map_text_values",
					map[string]any{
						"keys": []string{"country", "region", "locality"},
					},
				),
				"005": elementAssertion(
					"place_of_birth",
					"country",
					"mdoc.element_map_member_country_code",
					map[string]any{"member": "country"},
				),
				"006": elementAssertion(
					"place_of_birth",
					"region",
					"mdoc.element_map_member_utf8_max_length",
					map[string]any{
						"member": "region", "max_length": 150,
					},
				),
				"007": elementAssertion(
					"place_of_birth",
					"locality",
					"mdoc.element_map_member_utf8_max_length",
					map[string]any{
						"member": "locality", "max_length": 150,
					},
				),
			},
		},
		familySpec{
			marker: "IdentifyingData_Nationality_",
			claim:  "nationality",
			assertions: map[string]assertionSpec{
				"001": elementAssertion(
					"nationality",
					"present",
					"mdoc.namespace_element_present",
					nil,
				),
				"002": elementAssertion(
					"nationality",
					"strings",
					"mdoc.element_string_array",
					map[string]any{"min_items": 1},
				),
				"003": elementAssertion(
					"nationality",
					"country",
					"mdoc.element_country_code_array",
					map[string]any{"min_items": 1},
				),
				"004": elementAssertion(
					"nationality",
					"countries",
					"mdoc.element_country_code_array",
					map[string]any{"min_items": 1},
				),
			},
		},
		familySpec{
			marker: "IdentifyingData_Portrait_",
			claim:  "portrait",
			assertions: map[string]assertionSpec{
				"001": elementAssertion(
					"portrait",
					"present",
					"mdoc.namespace_element_present",
					nil,
				),
				"002": elementAssertion(
					"portrait",
					"bytes",
					"mdoc.element_cbor_type",
					map[string]any{"major_type": 2},
				),
				"003": elementAssertion("portrait", "jpeg", "mdoc.element_jpeg", nil),
			},
		},
		familySpec{
			marker: "IdentifyingData_Sex_",
			claim:  "sex",
			assertions: map[string]assertionSpec{
				"001": elementAssertion("sex", "present", "mdoc.namespace_element_present", nil),
				"002": elementAssertion(
					"sex",
					"unsigned",
					"mdoc.element_cbor_type",
					map[string]any{"major_type": 0},
				),
				"003": elementAssertion(
					"sex",
					"allowed",
					"mdoc.element_unsigned_integer_allowed",
					map[string]any{
						"allowed": []int{0, 1, 2, 3, 4, 5, 6, 9},
					},
				),
			},
		},
	)
	return specs
}

func claimAssertion(
	claim string,
	suffix string,
	validator string,
	extra map[string]any,
) assertionSpec {
	params := map[string]any{"claim": claim}
	for key, value := range extra {
		params[key] = value
	}
	return assertionSpec{
		id:        strings.NewReplacer(".", "_", "-", "_").Replace(claim) + "_" + suffix,
		validator: validator,
		params:    params,
	}
}

func elementAssertion(
	element string,
	suffix string,
	validator string,
	extra map[string]any,
) assertionSpec {
	params := map[string]any{
		"namespace": "eu.europa.ec.eudi.pid.1",
		"element":   element,
	}
	for key, value := range extra {
		params[key] = value
	}
	return assertionSpec{
		id:        strings.NewReplacer(".", "_", "-", "_").Replace(element) + "_" + suffix,
		validator: validator,
		params:    params,
	}
}

func matchSpec(id string, specs []familySpec) (familySpec, string, error) {
	matches := variantPattern.FindStringSubmatch(id)
	if matches == nil {
		return familySpec{}, "", fmt.Errorf("%s has no supported numeric variant", id)
	}
	for _, spec := range specs {
		if strings.Contains(id, spec.marker) {
			return spec, matches[1], nil
		}
	}
	return familySpec{}, "", fmt.Errorf("%s has no reviewed family mapping", id)
}

func readMarkdownSections(path string) (map[string][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	sections := map[string][]string{}
	current := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "## ") {
			current = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		if current != "" && line != "" {
			sections[current] = append(sections[current], line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return sections, nil
}

func cleanMarkdownLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "  ")
	return strings.TrimSpace(value)
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
