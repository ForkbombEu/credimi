// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package main generates the complete FCAF aggregate pipeline of every FCAF
// suite, plus the relying-party demo and happy-flow pipelines.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/forkbombeu/credimi/pkg/fcaf/aggregate"
)

// Suites are FCAF suite directories under the templates root, named
// <sut>/<role> as in the fcaf-validation `suite` input.
const (
	relyingPartySuite        = "wallet_solution/relying_party"
	attestationProviderSuite = "wallet_solution/attestation_provider"
)

// generatedSuites lists every suite whose scenarios make an aggregate pipeline.
var generatedSuites = []string{relyingPartySuite, attestationProviderSuite}

var demoScenarioNames = []string{
	"fcaf-wallet-solution-relying-party-engagement-haip-vp.yaml",
}

// demoTestIDs keeps the showcase bounded to one proven HAIP-VP happy flow.
var demoTestIDs = []string{
	"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_AddressData_Mobilephonenumber_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Mobilephonenumber_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Mobilephonenumber_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_AddressData_Residentaddress_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentaddress_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentcity_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentcity_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentcountry_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentcountry_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentcountry_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_AddressData_Residenthousenumber_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residenthousenumber_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentpostalcode_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentpostalcode_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentstate_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentstate_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_AddressData_Residentstreet_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_AddressData_Residentstreet_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Documentnumber_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Documentnumber_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Expirydate_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Expirydate_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Expirydate_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_Credentialmetadata_Expirydate_PID_IETF-sd-jwt-vc_004",
	"WS_RP_DM_Credentialmetadata_Issuancedate_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Issuancedate_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Issuancedate_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_Credentialmetadata_Issuancedate_PID_IETF-sd-jwt-vc_004",
	"WS_RP_DM_Credentialmetadata_Issuingauthority_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Issuingauthority_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Issuingcountry_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Issuingcountry_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Issuingcountry_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_Credentialmetadata_Issuingjurisdiction_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_Credentialmetadata_Issuingjurisdiction_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_Credentialmetadata_Issuingjurisdiction_PID_IETF-sd-jwt-vc_003",
	"WS_RP_DM_IdentifyingData_Familyname_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_IdentifyingData_Familyname_PID_IETF-sd-jwt-vc_002",
	"WS_RP_DM_IdentifyingData_Givenname_PID_IETF-sd-jwt-vc_001",
	"WS_RP_DM_IdentifyingData_Givenname_PID_IETF-sd-jwt-vc_002",
	"WS_RP_IA_Engagement__001",
	"WS_RP_SM_DeviceBinding__007",
	"WS_RP_SM_IssuerIntegrity__013",
}

// happyFlowScenarioNames keeps the happy flow to the shared-evidence
// scenarios that own the positive test batches; the fragmented one-test
// DCQL variants stay in the complete validation aggregate only.
var happyFlowScenarioNames = []string{
	"fcaf-wallet-solution-relying-party-engagement-haip-vp.yaml",
	"fcaf-wallet-solution-relying-party-pid-mdoc-data-model.yaml",
	"fcaf-wallet-solution-relying-party-pid-domestic.yaml",
	"fcaf-wallet-solution-relying-party-dcql-protocol-messages.yaml",
	"fcaf-wallet-solution-relying-party-direct-post-jwt-response-transport.yaml",
	"fcaf-wallet-solution-relying-party-dcql-baseline.yaml",
	"fcaf-wallet-solution-relying-party-dcql-main-interaction.yaml",
	"fcaf-wallet-solution-relying-party-dcql-rp-integrity.yaml",
	"fcaf-wallet-solution-relying-party-dcql-credential-formats.yaml",
	"fcaf-wallet-solution-relying-party-dcql-trust-mechanisms.yaml",
	"fcaf-wallet-solution-relying-party-dcql-session-encryption.yaml",
	"fcaf-wallet-solution-relying-party-dcql-interaction-metadata.yaml",
	"fcaf-wallet-solution-relying-party-request-object-by-value.yaml",
	"fcaf-wallet-solution-relying-party-dcql-credentials-match.yaml",
}

func main() {
	root := flag.String(
		"root",
		"config_templates/fcaf",
		"directory holding the FCAF suites as <sut>/<role>/{scenarios,tests,pipelines}",
	)
	only := flag.String(
		"suite",
		"",
		"generate one suite only, e.g. "+relyingPartySuite+"; empty generates every suite",
	)
	flag.Parse()

	suites := generatedSuites
	if *only != "" {
		if !slices.Contains(generatedSuites, *only) {
			fmt.Fprintf(os.Stderr, "unknown FCAF suite %q; known: %v\n", *only, generatedSuites)
			os.Exit(2)
		}
		suites = []string{*only}
	}
	for _, suite := range suites {
		if err := generateSuite(*root, suite); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

// generateSuite writes a suite's complete validation pipeline and, for the
// relying-party suite, its demo and happy-flow pipelines.
func generateSuite(root string, suite string) error {
	if err := generate(root, suite, suitePipelinePath(root, suite, "complete")); err != nil {
		return err
	}
	if suite != relyingPartySuite {
		return nil
	}
	if err := generateDemo(root, suitePipelinePath(root, suite, "demo")); err != nil {
		return err
	}
	return generateHappyFlow(root, suitePipelinePath(root, suite, "happy-flow"))
}

// suitePipelinePath is where a generated pipeline of the given kind lives,
// e.g. pipelines/fcaf-wallet-solution-relying-party-complete-validation.yaml.
func suitePipelinePath(root string, suite string, kind string) string {
	return filepath.Join(
		root,
		suite,
		"pipelines",
		"fcaf-"+aggregate.SuiteSlug(suite)+"-"+kind+"-validation.yaml",
	)
}

// suitePipelineName names a generated pipeline of the given kind, e.g.
// "FCAF wallet relying-party complete validation".
func suitePipelineName(suite string, kind string) string {
	sut, role, _ := strings.Cut(suite, "/")
	sut = strings.TrimSuffix(sut, "_solution")
	return fmt.Sprintf(
		"FCAF %s %s %s validation",
		strings.ReplaceAll(sut, "_", "-"),
		strings.ReplaceAll(role, "_", "-"),
		strings.ReplaceAll(kind, "-", " "),
	)
}

func generate(root string, suite string, outputPath string) error {
	paths, err := aggregate.ScenarioPaths(root, suite)
	if err != nil {
		return err
	}
	return write(root, paths, outputPath, aggregate.Options{
		Suite:   suite,
		Name:    suitePipelineName(suite, "complete"),
		NoVideo: new(true),
	})
}

func generateDemo(root string, outputPath string) error {
	selected, err := selectScenarios(root, relyingPartySuite, demoScenarioNames)
	if err != nil {
		return err
	}
	return write(root, selected, outputPath, aggregate.Options{
		Suite:   relyingPartySuite,
		Name:    suitePipelineName(relyingPartySuite, "demo"),
		TestIDs: demoTestIDs,
	})
}

// generateHappyFlow builds the aggregate pipeline validating every FCAF test
// owned by the happy-flow scenarios, without the curated demo filter.
func generateHappyFlow(root string, outputPath string) error {
	selected, err := selectScenarios(root, relyingPartySuite, happyFlowScenarioNames)
	if err != nil {
		return err
	}
	return write(root, selected, outputPath, aggregate.Options{
		Suite:                 relyingPartySuite,
		Name:                  suitePipelineName(relyingPartySuite, "happy-flow"),
		FilterTestsByEvidence: true,
		NoVideo:               new(true),
	})
}

func selectScenarios(root string, suite string, names []string) ([]string, error) {
	paths, err := aggregate.ScenarioPaths(root, suite)
	if err != nil {
		return nil, err
	}
	selected := make([]string, 0, len(names))
	for _, path := range paths {
		if slices.Contains(names, filepath.Base(path)) {
			selected = append(selected, path)
		}
	}
	if len(selected) != len(names) {
		return nil, fmt.Errorf(
			"FCAF scenarios incomplete: expected %v, found %d files",
			names,
			len(selected),
		)
	}
	return selected, nil
}

// write builds an aggregate pipeline and writes it to outputPath.
func write(root string, paths []string, outputPath string, opts aggregate.Options) error {
	built, err := aggregate.Build(root, paths, opts)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create aggregate pipeline directory: %w", err)
	}
	if err := os.WriteFile(outputPath, built.YAML, 0o644); err != nil {
		return fmt.Errorf("write aggregate FCAF pipeline: %w", err)
	}
	return nil
}
