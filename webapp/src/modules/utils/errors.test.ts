// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, it, expect } from 'vitest';

import { exceptionToError, getExceptionMessage } from './errors';

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
