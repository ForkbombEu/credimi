// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Canonical FCAF wallet-solution/relying-party grouping.
 *
 * Test identifiers encode their owning FCAF section and subsection directly:
 *
 *   WS_RP_<CATEGORY>_<SUBGROUP>_<specifics>_<NNN>
 *
 * e.g. `WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001`
 *      `WS_RP_IA_MainInteraction__003`
 *
 * `suite.section` in the test YAML is not reliable here: some entries carry the
 * semantic dotted path (`data_model.address_data`) while others carry ARF
 * clause references (`"6.1"`, `"3"`). The identifier prefix is stable across
 * every test, so it is the source of truth for grouping the report.
 */

export type FCAFCategoryCode = 'DM' | 'MS' | 'IA' | 'SM' | 'SH' | 'UC' | 'OTHER';

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

const CATEGORIES: Record<FCAFCategoryCode, FCAFCategory> = {
	DM: {
		code: 'DM',
		label: 'Data model',
		text: 'text-fcaf-data-model',
		bar: 'bg-fcaf-data-model',
		railBg: 'bg-fcaf-data-model/10',
		railBorder: 'border-fcaf-data-model'
	},
	MS: {
		code: 'MS',
		label: 'Message structure',
		text: 'text-fcaf-message-structure',
		bar: 'bg-fcaf-message-structure',
		railBg: 'bg-fcaf-message-structure/10',
		railBorder: 'border-fcaf-message-structure'
	},
	IA: {
		code: 'IA',
		label: 'Interaction',
		text: 'text-fcaf-interaction',
		bar: 'bg-fcaf-interaction',
		railBg: 'bg-fcaf-interaction/10',
		railBorder: 'border-fcaf-interaction'
	},
	SM: {
		code: 'SM',
		label: 'Security mechanisms',
		text: 'text-fcaf-security-mechanisms',
		bar: 'bg-fcaf-security-mechanisms',
		railBg: 'bg-fcaf-security-mechanisms/10',
		railBorder: 'border-fcaf-security-mechanisms'
	},
	SH: {
		code: 'SH',
		label: 'Shared',
		text: 'text-fcaf-shared',
		bar: 'bg-fcaf-shared',
		railBg: 'bg-fcaf-shared/10',
		railBorder: 'border-fcaf-shared'
	},
	UC: {
		code: 'UC',
		label: 'Use cases',
		text: 'text-fcaf-use-cases',
		bar: 'bg-fcaf-use-cases',
		railBg: 'bg-fcaf-use-cases/10',
		railBorder: 'border-fcaf-use-cases'
	},
	OTHER: {
		code: 'OTHER',
		label: 'Other',
		text: 'text-muted-foreground',
		bar: 'bg-muted-foreground',
		railBg: 'bg-muted/20',
		railBorder: 'border-muted-foreground/50'
	}
};

/** Display order for the report body. */
export const FCAF_CATEGORY_ORDER: FCAFCategoryCode[] = [
	'DM',
	'MS',
	'IA',
	'SM',
	'SH',
	'UC',
	'OTHER'
];

const SUBGROUP_LABELS: Record<string, string> = {
	addressdata: 'Address data',
	identifyingdata: 'Identifying data',
	credentialmetadata: 'Credential metadata',
	protocolmessages: 'Protocol messages',
	metadata: 'Metadata',
	credentialformats: 'Credential formats',
	maininteraction: 'Main interaction',
	engagement: 'Engagement',
	protocolflow: 'Protocol flow',
	supportive: 'Supportive',
	rpintegrity: 'RP integrity',
	trustmechanisms: 'Trust mechanisms',
	sessionencryption: 'Session encryption',
	devicebinding: 'Device binding',
	sessionbinding: 'Session binding',
	issuerintegrity: 'Issuer integrity',
	encoding: 'Encoding',
	cryptography: 'Cryptography',
	presentation: 'Presentation'
};

export type ParsedFCAFTestId = {
	category: FCAFCategory;
	/** Normalized subgroup key (lowercased segment), used as a stable map/loop key. */
	key: string;
	/** Human-readable subgroup label. */
	label: string;
};

export function subgroupLabel(segment: string): string {
	return SUBGROUP_LABELS[segment.toLowerCase()] ?? humanize(segment);
}

export function parseFCAFTestId(testId: string | undefined): ParsedFCAFTestId {
	if (!testId) return { category: CATEGORIES.OTHER, key: 'other', label: 'Other' };

	const parts = testId.split('_');
	// WS_RP_<CATEGORY>_<SUBGROUP>_...
	if (parts.length >= 4 && parts[0] === 'WS' && parts[1] === 'RP') {
		const code = parts[2].toUpperCase() as FCAFCategoryCode;
		const category = CATEGORIES[code] ?? CATEGORIES.OTHER;
		const segment = parts[3];
		return { category, key: segment.toLowerCase(), label: subgroupLabel(segment) };
	}

	return { category: CATEGORIES.OTHER, key: 'other', label: 'Other' };
}

function humanize(segment: string): string {
	const words = segment.replace(/([a-z0-9])([A-Z])/g, '$1 $2').split(' ');
	return words.map((word) => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
}
