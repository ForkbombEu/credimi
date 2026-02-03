// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { UsersResponse } from '@/pocketbase/types';

import { getUserDisplayName } from './utils';

describe('getUserDisplayName', () => {
	it('prefers the user name when available', () => {
		const user = {
			name: 'User Name',
			username: 'username',
			email: 'user@example.org'
		} as UsersResponse;

		expect(getUserDisplayName(user)).toBe('User Name');
	});

	it('falls back to username', () => {
		const user = {
			name: '',
			username: 'username',
			email: 'user@example.org'
		} as UsersResponse;

		expect(getUserDisplayName(user)).toBe('username');
	});

	it('falls back to email when no name or username', () => {
		const user = {
			name: '',
			username: '',
			email: 'user@example.org'
		} as UsersResponse;

		expect(getUserDisplayName(user)).toBe('user@example.org');
	});
});
