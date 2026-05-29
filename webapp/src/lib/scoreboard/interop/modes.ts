// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { m } from '@/i18n';

export const INTEROP_MODES = [
	'wallets_credentials',
	'wallets_issuers',
	'wallets_verifiers',
	'wallets_use_case_verifications',
	'wallets_conformance_checks',
	'use_case_verifications_conformance_checks'
] as const;

export type InteropMode = (typeof INTEROP_MODES)[number];

export const DEFAULT_INTEROP_MODE: InteropMode = 'wallets_credentials';

export function isInteropMode(value: string): value is InteropMode {
	return (INTEROP_MODES as readonly string[]).includes(value);
}

export function normalizeInteropMode(mode: string | null): InteropMode {
	return mode && isInteropMode(mode) ? mode : DEFAULT_INTEROP_MODE;
}

export function interopModeLabel(mode: InteropMode): string {
	switch (mode) {
		case 'wallets_credentials':
			return m.interop_mode_wallets_credentials();
		case 'wallets_issuers':
			return m.interop_mode_wallets_issuers();
		case 'wallets_verifiers':
			return m.interop_mode_wallets_verifiers();
		case 'wallets_use_case_verifications':
			return m.interop_mode_wallets_use_case_verifications();
		case 'wallets_conformance_checks':
			return m.interop_mode_wallets_conformance_checks();
		case 'use_case_verifications_conformance_checks':
			return m.interop_mode_use_case_verifications_conformance_checks();
	}
}

export type InteropModeTab = {
	value: InteropMode;
	label: () => string;
};

export function interopModeTabs(): InteropModeTab[] {
	return INTEROP_MODES.map((value) => ({
		value,
		label: () => interopModeLabel(value)
	}));
}
