// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Mirrors PocketBase default `~` behaviour for ordinary Hub search text:
 * case-insensitive substring contains (trim; empty query is not a hit).
 */
export function matchesHubSearchText(haystack: string | null | undefined, query: string): boolean {
	const needle = query.trim().toLowerCase();
	if (!needle) return false;
	if (haystack == null) return false;
	return haystack.toLowerCase().includes(needle);
}

/**
 * Soft-dim policy for nested Hub item chips:
 * - empty query → nothing dimmed (order unchanged)
 * - no nested name matches → nothing dimmed (order unchanged)
 * - at least one nested name matches → dim non-matches only; matches first
 */
export function annotateNestedHubItemsForSearch<T extends { name: string | null | undefined }>(
	items: readonly T[],
	query: string
): Array<T & { dimmed: boolean }> {
	const needle = query.trim();
	if (!needle) {
		return items.map((item) => ({ ...item, dimmed: false }));
	}

	const anyNestedMatch = items.some((item) => matchesHubSearchText(item.name, needle));
	if (!anyNestedMatch) {
		return items.map((item) => ({ ...item, dimmed: false }));
	}

	const annotated = items.map((item) => ({
		...item,
		dimmed: !matchesHubSearchText(item.name, needle)
	}));

	return annotated.sort((a, b) => Number(a.dimmed) - Number(b.dimmed));
}

/** Extract the active search string from CollectionManager / PocketBase query search options. */
export function resolveHubSearchQuery(search: unknown): string {
	const items = Array.isArray(search) ? search : search != null && search !== '' ? [search] : [];
	const first = items[0];
	if (typeof first === 'string') return first;
	if (first && typeof first === 'object' && 'text' in first) {
		const text = (first as { text: unknown }).text;
		if (typeof text === 'string') return text;
	}
	return '';
}
