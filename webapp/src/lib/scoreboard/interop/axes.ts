// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { m } from '@/i18n';

export const INTEROP_AXIS_KEYS = [
	'wallet',
	'issuer',
	'credential',
	'verifier',
	'use_case_verification',
	'conformance_check'
] as const;

export type InteropAxisKey = (typeof INTEROP_AXIS_KEYS)[number];

export function isInteropAxisKey(value: string): value is InteropAxisKey {
	return (INTEROP_AXIS_KEYS as readonly string[]).includes(value);
}

export function interopAxisLabel(axis: string): string {
	switch (axis) {
		case 'wallet':
			return m.Wallet();
		case 'issuer':
			return m.Issuer();
		case 'credential':
			return m.Credential();
		case 'verifier':
			return m.Verifier();
		case 'use_case_verification':
			return m.Use_case_verification();
		case 'conformance_check':
			return m.Conformance_check();
		default:
			return axis;
	}
}
