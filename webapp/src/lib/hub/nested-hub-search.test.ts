// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	annotateNestedHubItemsForSearch,
	matchesHubSearchText,
	resolveHubSearchQuery
} from './nested-hub-search';

describe('matchesHubSearchText', () => {
	it('returns false for empty or whitespace query', () => {
		expect(matchesHubSearchText('Passport', '')).toBe(false);
		expect(matchesHubSearchText('Passport', '   ')).toBe(false);
	});

	it('matches case-insensitively as substring', () => {
		expect(matchesHubSearchText('EUDI Passport', 'pass')).toBe(true);
		expect(matchesHubSearchText('EUDI Passport', 'PASSPORT')).toBe(true);
		expect(matchesHubSearchText('EUDI Passport', 'visa')).toBe(false);
	});

	it('returns false when haystack is nullish', () => {
		expect(matchesHubSearchText(null, 'pass')).toBe(false);
		expect(matchesHubSearchText(undefined, 'pass')).toBe(false);
	});
});

describe('annotateNestedHubItemsForSearch', () => {
	const items = [{ name: 'Passport' }, { name: 'Driving licence' }, { name: 'Student ID' }];

	it('dims nothing when query is empty', () => {
		expect(annotateNestedHubItemsForSearch(items, '')).toEqual([
			{ name: 'Passport', dimmed: false },
			{ name: 'Driving licence', dimmed: false },
			{ name: 'Student ID', dimmed: false }
		]);
	});

	it('dims nothing when no nested name matches (parent-only hit)', () => {
		expect(annotateNestedHubItemsForSearch(items, 'Acme Issuer')).toEqual([
			{ name: 'Passport', dimmed: false },
			{ name: 'Driving licence', dimmed: false },
			{ name: 'Student ID', dimmed: false }
		]);
	});

	it('dims non-matches when at least one nested name matches', () => {
		expect(annotateNestedHubItemsForSearch(items, 'pass')).toEqual([
			{ name: 'Passport', dimmed: false },
			{ name: 'Driving licence', dimmed: true },
			{ name: 'Student ID', dimmed: true }
		]);
	});

	it('puts matching nested items first while keeping relative order within groups', () => {
		expect(annotateNestedHubItemsForSearch(items, 'id')).toEqual([
			{ name: 'Student ID', dimmed: false },
			{ name: 'Passport', dimmed: true },
			{ name: 'Driving licence', dimmed: true }
		]);
	});

	it('treats nullish nested names as non-matches without throwing', () => {
		expect(
			annotateNestedHubItemsForSearch([{ name: null }, { name: 'Passport' }], 'pass')
		).toEqual([
			{ name: 'Passport', dimmed: false },
			{ name: null, dimmed: true }
		]);
	});
});

describe('resolveHubSearchQuery', () => {
	it('reads string and { text } search options', () => {
		expect(resolveHubSearchQuery(undefined)).toBe('');
		expect(resolveHubSearchQuery('foo')).toBe('foo');
		expect(resolveHubSearchQuery(['foo'])).toBe('foo');
		expect(resolveHubSearchQuery([{ text: 'bar', fields: ['name'] }])).toBe('bar');
	});
});
