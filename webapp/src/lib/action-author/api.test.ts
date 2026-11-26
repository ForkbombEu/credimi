// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { ClientResponseError } from 'pocketbase';
import { describe, expect, it, vi } from 'vitest';

const send = vi.fn();

vi.mock('@/pocketbase', () => ({
	pb: {
		send: (...args: unknown[]) => send(...args)
	}
}));

import { runFcafPipeline, SESSION_ID_PATTERN, startSession, ticketOf } from './api';

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

describe('runFcafPipeline', () => {
	it.each([
		[
			403,
			'forbidden',
			'only an owner of the organization can build and run its FCAF pipelines'
		],
		[
			422,
			'fcaf_credential_unavailable',
			'the FCAF scenarios use credentials or use cases your organization neither owns nor can use published: org/issuer/pid'
		]
	])('returns the endpoint message of a %i', async (status, reason, message) => {
		send.mockRejectedValueOnce(
			new ClientResponseError({
				status,
				response: { message, error: { code: status, reason, message } }
			})
		);
		const res = await runFcafPipeline('org/wallet/1-0', 'org/runner/device');
		expect(res.isErr).toBe(true);
		if (res.isErr) expect(res.error).toEqual({ status, reason, message });
	});
});

describe('startSession', () => {
	const base = { device_id: 'org/runner/device', wallet_version_identifier: 'org/wallet/1-0' };

	it.each([
		['chat', false, { ...base, mode: 'chat' }],
		['chat', true, { ...base, mode: 'chat' }],
		['fcaf', false, { ...base, mode: 'fcaf' }],
		['fcaf', true, { ...base, mode: 'fcaf', fcaf_fix: true }]
	] as const)('sends mode %s with fcafFix %s as %o', async (mode, fcafFix, body) => {
		send.mockResolvedValueOnce({ session_id: 's', ticket_id: 't', device_id: base.device_id });
		await startSession(base.device_id, base.wallet_version_identifier, mode, fcafFix);
		expect(send).toHaveBeenLastCalledWith(
			'/api/action-author/sessions',
			expect.objectContaining({ method: 'POST', body })
		);
	});
});
