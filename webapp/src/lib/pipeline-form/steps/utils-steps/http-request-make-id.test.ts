// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { httpRequestIdHost, httpRequestStepConfig } from './index.js';

describe('httpRequestIdHost', () => {
	it('uses the host for absolute URLs', () => {
		expect(httpRequestIdHost('https://example.com/path')).toBe('example.com');
	});

	it('does not throw on pipeline template URLs', () => {
		expect(
			httpRequestIdHost('${fixture.verifier_url}/openid4vp/sessions')
		).toBe('fixture-verifier-url-openid4vp-sessions');
	});

	it('falls back for empty input', () => {
		expect(httpRequestIdHost(undefined)).toBe('unknown');
		expect(httpRequestIdHost('')).toBe('unknown');
	});
});

describe('httpRequestStepConfig.makeId', () => {
	it('builds ids for absolute and template URLs', () => {
		expect(
			httpRequestStepConfig.makeId({
				method: 'POST',
				url: 'https://api.example.com/v1',
				body: ''
			})
		).toBe('http-post-api.example.com');

		expect(
			httpRequestStepConfig.makeId({
				method: 'GET',
				url: '${fixture.verifier_url}/sessions',
				body: ''
			})
		).toBe('http-get-fixture-verifier-url-sessions');
	});
});
