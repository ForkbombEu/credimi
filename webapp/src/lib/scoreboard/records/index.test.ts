// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ScoreboardRow } from '../types';

import { hasVisiblePipeline, PUBLISHED_PIPELINE_FILTER } from './index';

describe('scoreboard records visibility', () => {
	it('exposes a published-pipeline filter for public listings', () => {
		expect(PUBLISHED_PIPELINE_FILTER).toBe('pipeline.published = true');
	});

	it('treats rows with an expanded pipeline as visible', () => {
		const row = {
			expand: {
				pipeline: {
					id: 'pipe1',
					name: 'Capture Wallet Metadata Issue and Verification of credential',
					published: true
				}
			}
		} as ScoreboardRow;

		expect(hasVisiblePipeline(row)).toBe(true);
	});

	it('hides rows whose pipeline expand is missing (unpublished / private)', () => {
		const row = {
			id: 'cache1',
			pipeline: 'hidden-pipeline-id',
			expand: {}
		} as ScoreboardRow;

		expect(hasVisiblePipeline(row)).toBe(false);
	});
});
