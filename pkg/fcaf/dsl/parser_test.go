// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package dsl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDSLV2EmailPresentExample(t *testing.T) {
	def, err := Parse([]byte(`
id: WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001
title: Email address is present in the PID SD-JWT VC presentation
suite:
  sut: wallet_solution
  role: relying_party
  section: data_model.address_data.emailaddress
applicability:
  credential_format: ietf_sd_jwt_vc
  document_type: pid
normative_references:
  - title: ARF Annex 3.01 pid rulebook
    section: 4.2 Table 5
preconditions:
  - ref: pipeline.pid_sdjwt_presentation_success
  - ref: assertion.pid_sdjwt_all_required_and_ics_elements_requested
evidence:
  pid_sdjwt:
    from: pipeline.pid_sdjwt_presentation_success.outputs.pid_sdjwt
assertions:
  - id: email_present
    validator: sdjwt.claim_present
    input: evidence.pid_sdjwt
    params:
      claim: email
verdict:
  pass_when: all_assertions_pass
`))

	require.NoError(t, err)
	require.Equal(t, "wallet_solution", def.Suite.SUT)
	require.Equal(t, "pipeline.pid_sdjwt_presentation_success", def.Preconditions[0].Ref)
}

func TestParseDSLV2DependentEmailExample(t *testing.T) {
	def, err := Parse([]byte(`
id: WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_003
title: Email address is a UTF-8 string supporting the full Unicode range
suite:
  sut: wallet_solution
  role: relying_party
  section: data_model.address_data.emailaddress
applicability:
  credential_format: ietf_sd_jwt_vc
  document_type: pid
normative_references:
  - title: ARF Annex 3.01 pid rulebook
preconditions:
  - ref: pipeline.pid_sdjwt_presentation_success
  - ref: assertion.pid_sdjwt_all_required_and_ics_elements_requested
  - ref: test.WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001
evidence:
  pid_sdjwt:
    from: pipeline.pid_sdjwt_presentation_success.outputs.pid_sdjwt
assertions:
  - id: email_is_utf8_string
    validator: sdjwt.claim_utf8_string
    input: evidence.pid_sdjwt
    params:
      claim: email
verdict:
  pass_when: all_assertions_pass
`))

	require.NoError(t, err)
	require.Equal(
		t,
		"test.WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001",
		def.Preconditions[2].Ref,
	)
}

func TestParseRejectsDuplicateAssertionID(t *testing.T) {
	raw := `
id: test
title: duplicate assertions
suite:
  sut: wallet_solution
  role: relying_party
normative_references:
  - title: reference
preconditions:
  - ref: pipeline.pid_sdjwt
assertions:
  - id: email-present
    validator: sdjwt.claim_present
    input: evidence.pid_sdjwt
    params:
      claim: email
  - id: email-present
    validator: sdjwt.claim_present
    input: evidence.pid_sdjwt
    params:
      claim: email
verdict:
  pass_when: all_assertions_pass
`

	_, err := Parse([]byte(raw))
	require.ErrorContains(t, err, "duplicate assertion id")
}

func TestParsePipelinePreconditionRequiredSteps(t *testing.T) {
	def, err := ParsePrecondition([]byte(`
id: pipeline.shared
kind: pipeline
pipeline_id: owner/shared
required_steps:
  - issue-pid
  - present-pid
outputs:
  pid:
    path: $.output.present-pid.outputs
`))

	require.NoError(t, err)
	require.Equal(t, []string{"issue-pid", "present-pid"}, def.RequiredSteps)
}

func TestParsePipelinePreconditionRejectsInvalidRequiredSteps(t *testing.T) {
	_, err := ParsePrecondition([]byte(`
id: pipeline.shared
kind: pipeline
pipeline_id: owner/shared
required_steps:
  - present-pid
  - present-pid
outputs:
  pid:
    path: $.output.present-pid.outputs
`))

	require.ErrorContains(t, err, `duplicate required step "present-pid"`)
}
