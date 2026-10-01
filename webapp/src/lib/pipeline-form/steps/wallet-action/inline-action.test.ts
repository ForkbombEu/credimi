// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/canonify/index.js', () => ({
	getRecordByCanonifiedPath: vi.fn()
}));

vi.mock('$lib/hub', () => ({
	getHubItemById: vi.fn(),
	getHubItemLogo: vi.fn(),
	getHubItemUrl: vi.fn(() => '/hub/wallet')
}));

vi.mock('$lib', () => ({
	Wallet: {
		Action: {
			getCategoryLabel: () => 'Category'
		}
	}
}));

vi.mock('$lib/pipeline/device/query.js', () => ({
	findCachedDeviceByPath: vi.fn(async () => undefined)
}));

import { EXTERNAL_VERSION, GLOBAL_DEVICE } from '$pipeline-form/execution-target/types.js';

import { walletActionStepConfig } from './index.js';
import { isInlineWalletActionStepData } from './types.js';

describe('wallet-action inline action_code', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	const inlineYaml = {
		action_code: 'appId: eu.europa.ec.euidi\n---\n- tapOn: Share\n',
		version_id: EXTERNAL_VERSION,
		parameters: { DEEPLINK: '${{ prior.outputs.body.deeplink }}' }
	};

	it('deserializes action_code without network lookups for the action', async () => {
		const data = await walletActionStepConfig.deserialize(inlineYaml);

		expect(isInlineWalletActionStepData(data)).toBe(true);
		if (!isInlineWalletActionStepData(data)) return;

		expect(data.actionCode).toContain('tapOn: Share');
		expect(data.version).toBe(EXTERNAL_VERSION);
		expect(data.device).toBe(GLOBAL_DEVICE);
		expect(data.parameters).toEqual(inlineYaml.parameters);
	});

	it('round-trips action_code through serialize', async () => {
		const data = await walletActionStepConfig.deserialize(inlineYaml);
		expect(walletActionStepConfig.serialize(data)).toEqual({
			action_code: inlineYaml.action_code,
			version_id: EXTERNAL_VERSION,
			parameters: inlineYaml.parameters
		});
	});

	it('shows custom-code card metadata without exposing the script', async () => {
		const data = await walletActionStepConfig.deserialize(inlineYaml);
		const card = walletActionStepConfig.cardData(data);

		expect(card.title).toBe('Custom wallet action');
		expect(card.beforeTitle).toBe('Inline Maestro');
		expect(card.meta).toMatchObject({
			Code: 'Custom code'
		});
		expect(JSON.stringify(card)).not.toContain('tapOn');
	});

	it('makeId accepts action_code payloads', () => {
		expect(walletActionStepConfig.makeId(inlineYaml)).toBe('inline-maestro');
	});

	it('pass-through form preserves data on submit', async () => {
		const data = await walletActionStepConfig.deserialize(inlineYaml);
		const form = walletActionStepConfig.initForm({
			intent: 'edit',
			initial: data,
			getExecutionTarget: () => undefined,
			isExecutionTargetLocked: () => false
		});

		expect(form.canSave()).toBe(true);
		expect(form.getSubmitData()).toEqual(data);
		expect(walletActionStepConfig.serialize(form.getSubmitData()!)).toEqual({
			action_code: inlineYaml.action_code,
			version_id: EXTERNAL_VERSION,
			parameters: inlineYaml.parameters
		});
	});
});
