// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { isPrivatePage } from './private-pages';

describe('isPrivatePage', () => {
	it.each([
		['/my', true],
		['/my/', true],
		['/my/runners/dashboard-sign-in?runner=a/b&state=s', true],
		['/it/my', true],
		['/it/my/runners/dashboard-sign-in', true],
		['/pt-br/my/profile', true],
		['/my/__data.json', true],
		['/', false],
		['/it', false],
		['/myself', false],
		['/it/myself', false],
		['/login', false],
		['/hub/my', false],
		['/temporal-ui/namespaces', false],
		['/action-author-ui/', false]
	])('%s → %s', (path, privatePage) => {
		expect(isPrivatePage(new URL(path, 'https://credimi.example'))).toBe(privatePage);
	});
});
