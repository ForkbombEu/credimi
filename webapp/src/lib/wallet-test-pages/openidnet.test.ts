// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { getOpenIDConformanceWorkflowLogsProps, isOpenIDConformanceStandard } from './openidnet';

describe('OpenID conformance workflow logs', () => {
	it.each([
		['openid4vci_issuer', 'start-openid4vci-issuer-log-update'],
		['openid4vp_verifier', 'start-openid4vp-verifier-log-update'],
		['openid4vci_wallet', 'start-openidnet-check-log-update', '-log'],
		['openid4vp_wallet', 'start-openidnet-check-log-update', '-log']
	] as const)('selects the established adapter for %s', (standard, startSignal, suffix) => {
		const props = getOpenIDConformanceWorkflowLogsProps('workflow-1', 'tenant', standard);

		expect(props).toMatchObject({
			workflowId: 'workflow-1',
			namespace: 'tenant',
			subscriptionSuffix: 'openidnet-logs',
			startSignal
		});
		expect(props.workflowSignalSuffix).toBe(suffix);
	});

	it('recognizes only OpenID conformance standards with a log adapter', () => {
		expect(isOpenIDConformanceStandard('openid4vp_verifier')).toBe(true);
		expect(isOpenIDConformanceStandard('openid4vp_wallet')).toBe(true);
		expect(isOpenIDConformanceStandard('ewc')).toBe(false);
		expect(isOpenIDConformanceStandard(undefined)).toBe(false);
	});
});
