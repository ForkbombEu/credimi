// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * FCAF wallet-solution/relying-party grouping adapter.
 *
 * Labels and order come from `taxonomy.generated.ts` (generated from
 * `pkg/fcaf/taxonomy/taxonomy.json`). This module adds Tailwind colour tokens
 * and the ID parse helper. See `pkg/fcaf/taxonomy/README.md` for the grammar.
 */

import {
	FCAF_CATEGORY_LABELS,
	FCAF_CATEGORY_ORDER,
	FCAF_SUBGROUP_LABELS,
	type FCAFCategoryCode
} from './taxonomy.generated.js';

export type { FCAFCategoryCode };
export { FCAF_CATEGORY_ORDER };

export type FCAFCategory = {
	code: FCAFCategoryCode;
	label: string;
	// Static Tailwind classes backed by the --fcaf-* tokens in layout.css.
	// Kept as literal strings so Tailwind's content scanner emits them.
	text: string;
	bar: string;
	railBg: string;
	railBorder: string;
};

const CATEGORY_STYLES: Record<
	FCAFCategoryCode,
	Pick<FCAFCategory, 'text' | 'bar' | 'railBg' | 'railBorder'>
> = {
	DM: {
		text: 'text-fcaf-data-model',
		bar: 'bg-fcaf-data-model',
		railBg: 'bg-fcaf-data-model/10',
		railBorder: 'border-fcaf-data-model'
	},
	MS: {
		text: 'text-fcaf-message-structure',
		bar: 'bg-fcaf-message-structure',
		railBg: 'bg-fcaf-message-structure/10',
		railBorder: 'border-fcaf-message-structure'
	},
	IA: {
		text: 'text-fcaf-interaction',
		bar: 'bg-fcaf-interaction',
		railBg: 'bg-fcaf-interaction/10',
		railBorder: 'border-fcaf-interaction'
	},
	SM: {
		text: 'text-fcaf-security-mechanisms',
		bar: 'bg-fcaf-security-mechanisms',
		railBg: 'bg-fcaf-security-mechanisms/10',
		railBorder: 'border-fcaf-security-mechanisms'
	},
	SH: {
		text: 'text-fcaf-shared',
		bar: 'bg-fcaf-shared',
		railBg: 'bg-fcaf-shared/10',
		railBorder: 'border-fcaf-shared'
	},
	UC: {
		text: 'text-fcaf-use-cases',
		bar: 'bg-fcaf-use-cases',
		railBg: 'bg-fcaf-use-cases/10',
		railBorder: 'border-fcaf-use-cases'
	},
	OTHER: {
		text: 'text-muted-foreground',
		bar: 'bg-muted-foreground',
		railBg: 'bg-muted/20',
		railBorder: 'border-muted-foreground/50'
	}
};

function categoryFor(code: FCAFCategoryCode): FCAFCategory {
	return {
		code,
		label: FCAF_CATEGORY_LABELS[code],
		...CATEGORY_STYLES[code]
	};
}

const CATEGORIES = Object.fromEntries(
	FCAF_CATEGORY_ORDER.map((code) => [code, categoryFor(code)])
) as Record<FCAFCategoryCode, FCAFCategory>;

export type ParsedFCAFTestId = {
	category: FCAFCategory;
	/** Normalized subgroup key (lowercased segment), used as a stable map/loop key. */
	key: string;
	/** Human-readable subgroup label. */
	label: string;
};

export function subgroupLabel(segment: string): string {
	return FCAF_SUBGROUP_LABELS[segment.toLowerCase()] ?? humanize(segment);
}

export function parseFCAFTestId(testId: string | undefined): ParsedFCAFTestId {
	if (!testId) return { category: CATEGORIES.OTHER, key: 'other', label: 'Other' };

	const parts = testId.split('_');
	// WS_RP_<CATEGORY>_<SUBGROUP>_...
	if (parts.length >= 4 && parts[0] === 'WS' && parts[1] === 'RP') {
		const code = parts[2].toUpperCase() as FCAFCategoryCode;
		const category = CATEGORIES[code];
		if (category) {
			const segment = parts[3];
			return { category, key: segment.toLowerCase(), label: subgroupLabel(segment) };
		}
	}

	return { category: CATEGORIES.OTHER, key: 'other', label: 'Other' };
}

function humanize(segment: string): string {
	const words = segment.replace(/([a-z0-9])([A-Z])/g, '$1 $2').split(' ');
	return words.map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
}
