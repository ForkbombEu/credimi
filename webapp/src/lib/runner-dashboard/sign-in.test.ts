// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

import { loginReturnPath } from '@/auth/login-return';

vi.mock('@/pocketbase', () => ({ pb: {} }));

import { dashboardHost } from './sign-in';

describe('dashboardHost', () => {
	it.each([
		['http://192.168.1.20:8051/auth/credimi/callback', '192.168.1.20:8051'],
		['https://runner.example.org/auth/credimi/callback', 'runner.example.org'],
		['ftp://runner.example.org/auth/credimi/callback', undefined],
		['javascript:alert(1)', undefined],
		['/auth/credimi/callback', undefined],
		['', undefined]
	])('%s → %s', (returnTo, host) => {
		expect(dashboardHost(returnTo)).toBe(host);
	});
});

describe('loginReturnPath', () => {
	it.each([
		[
			'/my/runners/dashboard-sign-in?runner=a/b&state=s',
			'/my/runners/dashboard-sign-in?runner=a/b&state=s'
		],
		['/my', '/my'],
		['/myself', undefined],
		['/login', undefined],
		['//evil.example/my/', undefined],
		['/\\evil.example/my/', undefined],
		['https://evil.example/my/', undefined],
		['/my/../login', undefined]
	])('%s → %s', (path, expected) => {
		expect(loginReturnPath(path)).toBe(expected);
	});
});
