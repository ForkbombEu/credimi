// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { parseFCAFTestId } from './categories';

describe('parseFCAFTestId', () => {
	it('maps DM identifiers to the data model category', () => {
		const { category, key, label } = parseFCAFTestId(
			'WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001'
		);
		expect(category.code).toBe('DM');
		expect(category.label).toBe('Data model');
		expect(key).toBe('addressdata');
		expect(label).toBe('Address data');
	});

	it('handles double-underscore interaction identifiers', () => {
		const { category, label } = parseFCAFTestId('WS_RP_IA_MainInteraction__003');
		expect(category.code).toBe('IA');
		expect(label).toBe('Main interaction');
	});

	it('normalizes credential metadata casing variants', () => {
		expect(parseFCAFTestId('WS_RP_DM_Credentialmetadata_X_001').label).toBe(
			'Credential metadata'
		);
		expect(parseFCAFTestId('WS_RP_DM_CredentialMetadata_X_001').label).toBe(
			'Credential metadata'
		);
	});

	it('maps RpIntegrity to the RP integrity label', () => {
		expect(parseFCAFTestId('WS_RP_SM_RpIntegrity_ABC_001').label).toBe('RP integrity');
	});

	it('groups attestation-provider identifiers like relying-party ones', () => {
		const { category, key, label } = parseFCAFTestId('WS_AP_IA_Supportive_010a');
		expect(category.code).toBe('IA');
		expect(key).toBe('supportive');
		expect(label).toBe('Supportive');
		expect(parseFCAFTestId('WS_XX_IA_Supportive_010a').category.code).toBe('OTHER');
	});

	it('falls back to Other for unknown prefixes', () => {
		const { category, label } = parseFCAFTestId('SOMETHING_ELSE');
		expect(category.code).toBe('OTHER');
		expect(label).toBe('Other');
	});

	it('falls back to Other for missing identifiers', () => {
		expect(parseFCAFTestId(undefined).category.code).toBe('OTHER');
		expect(parseFCAFTestId('').category.code).toBe('OTHER');
	});
});
