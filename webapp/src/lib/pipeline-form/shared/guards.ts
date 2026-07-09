// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import {
	EXTERNAL_VERSION,
	GLOBAL_RUNNER,
	type MobileTargetFields,
	type SelectedRunner,
	type SelectedVersion
} from './mobile-target.js';

//

function isHubItemLike(value: unknown): value is HubItem {
	if (!value || typeof value !== 'object') return false;
	const obj = value as Record<string, unknown>;
	return typeof obj.id === 'string' && typeof obj.name === 'string';
}

function isSelectedVersion(value: unknown): value is SelectedVersion {
	if (value === EXTERNAL_VERSION) return true;
	if (!value || typeof value !== 'object') return false;
	const obj = value as Record<string, unknown>;
	return typeof obj.id === 'string' && typeof obj.tag === 'string';
}

function isSelectedRunner(value: unknown): value is SelectedRunner {
	if (value === GLOBAL_RUNNER) return true;
	if (!value || typeof value !== 'object') return false;
	const obj = value as Record<string, unknown>;
	return typeof obj.name === 'string' && typeof obj.path === 'string';
}

export function isMobileTargetFields(value: unknown): value is MobileTargetFields {
	if (!value || typeof value !== 'object') return false;
	const obj = value as Record<string, unknown>;
	return (
		'wallet' in obj &&
		isHubItemLike(obj.wallet) &&
		'version' in obj &&
		isSelectedVersion(obj.version) &&
		'runner' in obj &&
		isSelectedRunner(obj.runner)
	);
}
