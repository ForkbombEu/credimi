// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/** Standards and suites Credimi runs, as listed on the marketing pages. */
export const standards: { name: string; scope: string }[] = [
	{ name: 'OpenID4VCI 1.0', scope: 'Issuer and wallet' },
	{ name: 'OpenID4VP 1.0, Draft 23, Draft 24', scope: 'Verifier and wallet' },
	{ name: 'OpenID Foundation conformance suite', scope: 'Integrated' },
	{ name: 'WE BUILD / EWC conformance', scope: 'Integrated' },
	{ name: 'EUDIW verifier tests', scope: 'Integrated' },
	{ name: 'SD-JWT VC and ISO mDoc PID', scope: 'Credential formats' },
	{ name: 'DCQL and Digital Credentials API', scope: 'Scenarios' }
];

/** EUDI FCAF catalog size; update together with the date below. */
export const fcafCatalog = { testCases: 617, scenarios: 184, asOf: '20/10/2026' };
