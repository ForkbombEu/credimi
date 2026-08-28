// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package dsl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDirectPipelineEvidenceDefinition(t *testing.T) {
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
evidence:
  pid_sdjwt:
    from: pipeline.pid.presentation.sdjwt.all-claims.outputs.pid_sdjwt
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
	require.Equal(
		t,
		"pipeline.pid.presentation.sdjwt.all-claims.outputs.pid_sdjwt",
		def.Evidence["pid_sdjwt"].From,
	)
}

func TestParseRejectsLegacyEvidenceBinding(t *testing.T) {
	_, err := Parse([]byte(`
id: test
suite:
  sut: wallet_solution
  role: relying_party
normative_references:
  - title: reference
evidence:
  pid_sdjwt:
    from: preconditions.pipeline.pid.outputs.pid_sdjwt
assertions:
  - id: email-present
    validator: sdjwt.claim_present
    input: evidence.pid_sdjwt
verdict:
  pass_when: all_assertions_pass
`))

	require.ErrorContains(t, err, "must match pipeline.<id>.outputs.<name>")
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
evidence:
  pid_sdjwt:
    from: pipeline.pid.outputs.pid_sdjwt
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
