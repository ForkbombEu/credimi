// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { jsonParseStepConfig } from '.';
import { getConfigByType, getDisplayData } from '../..';

const data = {
	rawJSON: '{"dcql_query":{"credentials":[]}}',
	struct_type: 'map'
};

describe('json-parse step config', () => {
	it('is registered for pipeline editing', () => {
		expect(getConfigByType('json-parse')).toBe(jsonParseStepConfig);
		expect(getDisplayData('json-parse').labels.singular).toBe('JSON Parse');
	});

	it('round-trips parse configuration', async () => {
		const formData = await jsonParseStepConfig.deserialize(data);

		expect(jsonParseStepConfig.serialize(formData)).toEqual(data);
	});

	it('falls back to defaults when deserializing empty input', async () => {
		const formData = await jsonParseStepConfig.deserialize({
			rawJSON: '',
			struct_type: ''
		});

		expect(jsonParseStepConfig.serialize(formData)).toEqual({
			rawJSON: '{}',
			struct_type: 'map'
		});
	});

	it('shows struct type on card', async () => {
		const formData = await jsonParseStepConfig.deserialize(data);

		expect(jsonParseStepConfig.cardData(formData)).toEqual({
			title: 'JSON Parse',
			copyText: 'map'
		});
	});

	it('derives an id from struct type', () => {
		expect(jsonParseStepConfig.makeId(data)).toBe('json-parse-map');
	});
});
