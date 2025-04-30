// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import { z } from 'zod';
import type { StandardWithTestSuites } from './_partials/logic';
import { m } from '@/i18n';

//

const variantsSchema = z.object({
	variants: z.array(z.string())
});

export const load = async ({ fetch }) => {
	const openid4vpVariantsResponse = await pb.send('/api/template/blueprints', {
		method: 'GET',
		fetch
	});

	const openid4vciVariantsResponse = await pb.send(
		'/api/template/blueprints?test_id=OpenID4VCI_Wallet/eudiw',
		{
			method: 'GET',
			fetch
		}
	);

	const parseOpenid4vpResult = variantsSchema.safeParse(openid4vpVariantsResponse);
	const parseOpenid4vciResult = variantsSchema.safeParse(openid4vciVariantsResponse);

	if (!parseOpenid4vpResult.success || !parseOpenid4vciResult.success) {
		throw new Error('Failed to parse variants response');
	}

	const standardsAndTestSuites: StandardWithTestSuites[] = [
		{
			id: 'OpenID4VP_Wallet',
			label: 'OpenID4VP Wallet',
			description: m.openid4vp_wallet_standard_description(),
			testSuites: [
				{
					id: 'OpenID_Foundation',
					label: 'OpenID4VP Wallet',
					tests: parseOpenid4vpResult.data.variants
				}
			]
		},
		{
			id: 'OpenID4VP_Verifier',
			label: 'OpenID4VP Verifier',
			description: m.openid4vp_verifier_standard_description(),
			testSuites: [
				{
					id: 'OpenID_Foundation',
					label: 'OpenID4VP Verifier',
					tests: parseOpenid4vciResult.data.variants
				}
			],
			disabled: true
		},
		{
			id: 'OpenID4VCI_Wallet',
			label: 'OpenID4VCI Wallet',
			description: m.openid4vci_wallet_standard_description(),
			testSuites: [
				{
					id: 'eudiw',
					label: 'OpenID4VCI Wallet',
					tests: parseOpenid4vciResult.data.variants
				}
			]
		},
		{
			id: 'OpenID4VCI_Issuer',
			label: 'OpenID4VCI Issuer',
			description: m.openid4vci_issuer_standard_description(),
			testSuites: [
				{
					id: 'eudiw',
					label: 'OpenID4VCI Issuer',
					tests: parseOpenid4vciResult.data.variants
				}
			],
			disabled: true
		},
		{
			id: 'VC-API-Issuer',
			label: 'VC-API Issuer',
			description: m.vc_api_issuer_standard_description(),
			testSuites: [
				{
					id: 'eudiw',
					label: 'OpenID4VCI Issuer',
					tests: parseOpenid4vciResult.data.variants
				}
			],
			disabled: true
		},
		{
			id: 'VC-API-Verifier',
			label: 'VC-API Verifier',
			description: m.vc_api_verifier_standard_description(),
			testSuites: [
				{
					id: 'eudiw',
					label: 'OpenID4VCI Issuer',
					tests: parseOpenid4vciResult.data.variants
				}
			],
			disabled: true
		}
	];

	return {
		standardsAndTestSuites
	};
};
