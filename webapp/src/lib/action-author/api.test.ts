// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { SESSION_ID_PATTERN, ticketOf } from './api';

const TICKET = '0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d';
const SESSION_ID = `ActionAuthorSession-abcdefghij12345-${TICKET}`;

describe('ticketOf', () => {
	it('returns the ticket of a session ID', () => {
		expect(SESSION_ID_PATTERN.test(SESSION_ID)).toBe(true);
		expect(ticketOf(SESSION_ID)).toBe(TICKET);
	});

	it('throws on another ID', () => {
		expect(SESSION_ID_PATTERN.test(`ActionAuthorSession-${TICKET}`)).toBe(false);
		expect(() => ticketOf(`ActionAuthorSession-${TICKET}`)).toThrow(
			'not an action author session id'
		);
	});
});
