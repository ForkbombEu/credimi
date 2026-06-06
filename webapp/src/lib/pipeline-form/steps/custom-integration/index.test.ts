// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { CustomChecksResponse } from '@/pocketbase/types';

vi.mock('$lib/canonify/index.js', () => ({
	getRecordByCanonifiedPath: vi.fn()
}));

vi.mock('$lib/utils', () => ({
	getPath: vi.fn((record: { canonified_name?: string }, trim?: boolean) =>
		trim ? (record.canonified_name ?? '') : (record.canonified_name ?? '')
	)
}));

vi.mock('@/i18n/index.js', async (importOriginal) => {
	const actual = await importOriginal<typeof import('@/i18n/index.js')>();
	return {
		...actual,
		m: {
			...actual.m,
			Pipeline_form_missing_check_id: () => 'Missing check ID'
		}
	};
});

vi.mock('$lib/hub/utils.js', () => ({
	getCustomCheckPublicUrl: (item: { canonified_name?: string }) =>
		`/my/custom-integrations/${item.canonified_name ?? ''}/run`
}));

import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';

import { customCheckStepConfig } from './index.js';

const integration = {
	id: 'ci1',
	name: 'My Integration',
	canonified_name: 'org/my-integration',
	logo: 'logo.png'
} as CustomChecksResponse;

describe('customCheckStepConfig', () => {
	beforeEach(() => {
		vi.mocked(getRecordByCanonifiedPath).mockReset();
	});

	it('serialize emits check_id only when config is absent', () => {
		expect(customCheckStepConfig.serialize({ integration })).toEqual({
			check_id: 'org/my-integration'
		});
	});

	it('serialize includes config when non-empty', () => {
		expect(
			customCheckStepConfig.serialize({
				integration,
				config: { apiKey: 'secret' }
			})
		).toEqual({
			check_id: 'org/my-integration',
			config: { apiKey: 'secret' }
		});
	});

	it('serialize omits config when empty object', () => {
		expect(
			customCheckStepConfig.serialize({
				integration,
				config: {}
			})
		).toEqual({
			check_id: 'org/my-integration'
		});
	});

	it('deserialize loads integration and config', async () => {
		vi.mocked(getRecordByCanonifiedPath).mockResolvedValue(integration);

		const result = await customCheckStepConfig.deserialize({
			check_id: 'org/my-integration',
			config: { apiKey: 'secret' }
		});

		expect(getRecordByCanonifiedPath).toHaveBeenCalledWith('org/my-integration');
		expect(result).toEqual({
			integration,
			config: { apiKey: 'secret' }
		});
	});

	it('deserialize throws when check_id is missing', async () => {
		await expect(customCheckStepConfig.deserialize({})).rejects.toThrow('Missing check ID');
	});

	it('deserialize propagates lookup errors', async () => {
		const err = new Error('not found');
		vi.mocked(getRecordByCanonifiedPath).mockResolvedValue(err);

		await expect(
			customCheckStepConfig.deserialize({ check_id: 'org/missing' })
		).rejects.toThrow('not found');
	});

	it('cardData uses integration name and public run URL', () => {
		const card = customCheckStepConfig.cardData({ integration });
		expect(card.title).toBe('My Integration');
		expect(card.copyText).toBe('org/my-integration');
		expect(card.publicUrl).toContain('/my/custom-integrations/org/my-integration/run');
	});
});
