// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { IndexItem } from '$lib/layout/pageIndex.svelte';

import { Building2, Code, FolderCheck, Key, Layers, Layers3, QrCode, ScanEye } from 'lucide-svelte';

import { m } from '@/i18n';

// import type { MarketplaceItemType } from './utils';

// export type MarketplacePageType = Exclude<MarketplaceItemType, 'custom_checks'>;

// export interface SectionOptions {
// 	hasDescription?: boolean;
// 	hasCredentials?: boolean;
// 	hasUseCaseVerifications?: boolean;
// 	hasRelatedVerifier?: boolean;
// 	hasRelatedCredentials?: boolean;
// 	hasCompatibleIssuer?: boolean;
// 	hasConformanceChecks?: boolean;
// 	hasActions?: boolean;
// }

// type SectionDefinition = {
// 	icon: IconComponent;
// 	anchor: string;
// 	label: () => string;
// 	condition?: keyof SectionOptions;
// };

// type PageSectionConfig = {
// 	sections: SectionDefinition[];
// };

export const sections = {
	general_info: {
		icon: Building2,
		anchor: 'general_info',
		label: m.General_info()
	},
	credential_properties: {
		icon: Building2,
		anchor: 'credential_properties',
		label: m.Credential_properties()
	},
	description: {
		icon: Layers,
		anchor: 'description',
		label: m.Description()
	},
	credential_subjects: {
		icon: Layers3,
		anchor: 'credential_subject',
		label: m.Credential_subject()
	},
	compatible_issuer: {
		icon: FolderCheck,
		anchor: 'compatible_issuer',
		label: m.Compatible_issuer()
	},
	credentials: {
		icon: Layers,
		anchor: 'credentials',
		label: m.Supported_credentials()
	},
	linked_credentials: {
		icon: Key,
		anchor: 'credentials',
		label: m.Linked_credentials()
	},
	use_case_verifications: {
		icon: ScanEye,
		anchor: 'use_case_verifications',
		label: m.Use_case_verifications()
	},
	related_verifier: {
		icon: Layers3,
		anchor: 'related_verifier',
		label: m.Related_verifier()
	},
	related_credentials: {
		icon: Key,
		anchor: 'related_credentials',
		label: m.Related_credentials()
	},
	conformance_checks: {
		icon: Layers3,
		anchor: 'conformance_checks',
		label: m.Conformance_Checks()
	},
	actions: {
		icon: Code,
		anchor: 'actions',
		label: m.Actions()
	},
	qr_code: {
		icon: QrCode,
		anchor: 'qr_code',
		label: m.QR_code()
	},
	workflow_yaml: {
		icon: Code,
		anchor: 'workflow_yaml',
		label: m.Workflow_YAML()
	}
} satisfies Record<string, IndexItem>;

// const PAGE_CONFIGURATIONS: Record<MarketplacePageType, PageSectionConfig> = {
// 	credentials: {
// 		sections: [
// 			SECTION_DEFINITIONS.credential_properties,
// 			SECTION_DEFINITIONS.description,
// 			SECTION_DEFINITIONS.credential_subjects,
// 			SECTION_DEFINITIONS.compatible_issuer
// 		]
// 	},
// 	credential_issuers: {
// 		sections: [
// 			SECTION_DEFINITIONS.general_info,
// 			SECTION_DEFINITIONS.description,
// 			SECTION_DEFINITIONS.credentials
// 		]
// 	},
// 	verifiers: {
// 		sections: [
// 			SECTION_DEFINITIONS.general_info,
// 			SECTION_DEFINITIONS.description,
// 			SECTION_DEFINITIONS.linked_credentials,
// 			SECTION_DEFINITIONS.use_case_verifications
// 		]
// 	},
// 	use_cases_verifications: {
// 		sections: [
// 			SECTION_DEFINITIONS.general_info,
// 			SECTION_DEFINITIONS.related_verifier,
// 			SECTION_DEFINITIONS.related_credentials
// 		]
// 	},
// 	wallets: {
// 		sections: [
// 			SECTION_DEFINITIONS.general_info,
// 			SECTION_DEFINITIONS.description,
// 			SECTION_DEFINITIONS.conformance_checks,
// 			SECTION_DEFINITIONS.actions
// 		]
// 	}
// };

// export function generateMarketplaceSection(
// 	pageType: MarketplacePageType,
// 	options: SectionOptions = {}
// ): Record<string, IndexItem> {
// 	const config = PAGE_CONFIGURATIONS[pageType];
// 	if (!config) {
// 		throw new Error(`Unknown marketplace page type: ${pageType}`);
// 	}

// 	const sections: Record<string, IndexItem> = {};
// 	for (const sectionDef of config.sections) {
// 		if (sectionDef.condition && !options[sectionDef.condition]) {
// 			continue;
// 		}
// 		const sectionKey = getSectionKey(sectionDef);
// 		sections[sectionKey] = {
// 			icon: sectionDef.icon,
// 			anchor: sectionDef.anchor,
// 			label: sectionDef.label()
// 		};
// 	}

// 	return sections;
// }

// function getSectionKey(sectionDef: SectionDefinition): string {
// 	for (const [key, def] of Object.entries(SECTION_DEFINITIONS)) {
// 		if (def === sectionDef) {
// 			// Convert some special cases for backwards compatibility
// 			if (key === 'linked_credentials') return 'credentials';
// 			return key;
// 		}
// 	}
// 	return sectionDef.anchor;
// }
