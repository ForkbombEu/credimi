// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { EXTERNAL_VERSION, GLOBAL_DEVICE } from '$pipeline-form/execution-target/types.js';

import { applyWalletActionStepVersion, isMatchingMobileStepData } from './change-wallet-version.js';
import type { InlineWalletActionStepData, StoredWalletActionStepData } from './types.js';

describe('Matching mobile steps / Change wallet version helpers', () => {
	const stored: StoredWalletActionStepData = {
		kind: 'stored',
		wallet: { id: 'w-a', name: 'Wallet A' } as never,
		version: EXTERNAL_VERSION,
		device: GLOBAL_DEVICE,
		action: { id: 'a1', name: 'Action' } as never
	};

	const inline: InlineWalletActionStepData = {
		kind: 'inline',
		actionCode: 'appId: x\n---\n- tapOn: Share\n',
		version: EXTERNAL_VERSION,
		device: GLOBAL_DEVICE
	};

	const nextVersion = {
		id: 'v2',
		tag: '2.0',
		__canonified_path__: 'org/w-a/v2'
	} as StoredWalletActionStepData['version'];

	it('treats only stored wallet-action data as Matching mobile step data', () => {
		expect(isMatchingMobileStepData(stored)).toBe(true);
		expect(isMatchingMobileStepData(inline)).toBe(false);
		expect(isMatchingMobileStepData(null)).toBe(false);
		expect(isMatchingMobileStepData({})).toBe(false);
	});

	it('applies Step version only for stored Matching mobile steps of that wallet', () => {
		expect(applyWalletActionStepVersion(inline, 'w-a', nextVersion)).toBeUndefined();
		expect(applyWalletActionStepVersion(stored, 'w-other', nextVersion)).toBeUndefined();

		const updated = applyWalletActionStepVersion(stored, 'w-a', nextVersion);
		expect(updated).toEqual({ ...stored, version: nextVersion });
		expect(stored.version).toBe(EXTERNAL_VERSION);
	});
});
