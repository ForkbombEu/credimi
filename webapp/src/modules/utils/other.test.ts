// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { capitalize, ensureArray, maybeArrayIsValue, removeTrailingSlash } from './other';

describe('other utils', () => {
	it('capitalizes the first character', () => {
		expect(capitalize('hello')).toBe('Hello');
		expect(capitalize('')).toBe('');
	});

	it('removes trailing slashes', () => {
		expect(removeTrailingSlash('path/')).toBe('path');
		expect(removeTrailingSlash('path')).toBe('path');
	});

	it('ensures array values', () => {
		expect(ensureArray('value')).toEqual(['value']);
		expect(ensureArray(['value', 'next'])).toEqual(['value', 'next']);
		expect(ensureArray(undefined)).toEqual([]);
	});

	it('detects whether maybe arrays contain values', () => {
		expect(maybeArrayIsValue(['value'])).toBe(true);
		expect(maybeArrayIsValue([])).toBe(false);
		expect(maybeArrayIsValue('value')).toBe(true);
		expect(maybeArrayIsValue(undefined)).toBe(false);
	});
});
