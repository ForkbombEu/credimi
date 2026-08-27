// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';

import { exceptionToError, getExceptionMessage, localizePocketBaseErrorCode } from './errors';

describe('errors utils', () => {
	it('returns the message for Error instances', () => {
		const err = new Error('boom');
		expect(getExceptionMessage(err)).toBe('boom');
	});

	it('stringifies non-error values', () => {
		expect(getExceptionMessage({ code: 42 })).toBe('{"code":42}');
	});

	it('wraps non-error values into Error', () => {
		const err = exceptionToError('nope');
		expect(err).toBeInstanceOf(Error);
		expect(err.message).toBe('Unexpected error: "nope"');
	});

	it('returns Error values as-is', () => {
		const err = new Error('keep');
		expect(exceptionToError(err)).toBe(err);
	});
});

describe('localizePocketBaseErrorCode', () => {
	const catalog = {
		validation_wallet_action_market_link_requires_install_app: () => 'Localized wallet error',
		validation_not_a_function: 'oops',
		validation_throws: () => {
			throw new Error('needs inputs');
		},
		validation_non_string: () => 42
	} as Record<string, unknown>;

	it('returns localized string when code matches a message function', () => {
		expect(
			localizePocketBaseErrorCode(
				'validation_wallet_action_market_link_requires_install_app',
				'fallback',
				catalog
			)
		).toBe('Localized wallet error');
	});

	it('returns fallback for unknown codes', () => {
		expect(localizePocketBaseErrorCode('validation_unknown', 'fallback', catalog)).toBe(
			'fallback'
		);
	});

	it('returns fallback when code is undefined', () => {
		expect(localizePocketBaseErrorCode(undefined, 'fallback', catalog)).toBe('fallback');
	});

	it('returns fallback when catalog value is not a function', () => {
		expect(localizePocketBaseErrorCode('validation_not_a_function', 'fallback', catalog)).toBe(
			'fallback'
		);
	});

	it('returns fallback when message function throws', () => {
		expect(localizePocketBaseErrorCode('validation_throws', 'fallback', catalog)).toBe(
			'fallback'
		);
	});

	it('returns fallback when message function returns a non-string', () => {
		expect(localizePocketBaseErrorCode('validation_non_string', 'fallback', catalog)).toBe(
			'fallback'
		);
	});
});
