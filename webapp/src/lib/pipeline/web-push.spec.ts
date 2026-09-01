// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { urlBase64ToUint8Array } from './web-push';

describe('urlBase64ToUint8Array', () => {
	it('decodes base64 without padding', () => {
		// atob('AQID') = bytes 0x01 0x02 0x03
		expect(urlBase64ToUint8Array('AQID')).toEqual(new Uint8Array([1, 2, 3]));
	});

	it('adds missing padding', () => {
		// 'AQ' must be padded to 'AQ==' before decoding
		expect(urlBase64ToUint8Array('AQ')).toEqual(new Uint8Array([1]));
	});

	it('converts URL-safe characters', () => {
		// '+' (62) and '/' (63) are sent as '-' and '_' in base64url
		expect(urlBase64ToUint8Array('-_')).toEqual(new Uint8Array([0xfb]));
	});

	it('round-trips a 65-byte VAPID key', () => {
		const raw = crypto.getRandomValues(new Uint8Array(65));
		const encoded = Buffer.from(raw).toString('base64url');
		expect(Array.from(urlBase64ToUint8Array(encoded))).toEqual(Array.from(raw));
	});
});
