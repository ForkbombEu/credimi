// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';
import type { Record as RunnerRecord } from '$lib/pipeline/runner';

import type { WalletVersionsResponse } from '@/pocketbase/types';

//

export const GLOBAL_RUNNER = 'global' as const;
export const EXTERNAL_VERSION = 'installed_from_external_source' as const;

export type SelectedRunner = RunnerRecord | typeof GLOBAL_RUNNER;
export type SelectedVersion = WalletVersionsResponse | typeof EXTERNAL_VERSION;

export type ExecutionTarget = {
	wallet: HubItem;
	version: SelectedVersion;
	runner: SelectedRunner;
};

function isHubItemLike(value: unknown): value is HubItem {
	if (!value || typeof value !== 'object') return false;
	const obj = value as { id?: unknown; name?: unknown };
	return typeof obj.id === 'string' && typeof obj.name === 'string';
}

function isSelectedVersion(value: unknown): value is SelectedVersion {
	if (value === EXTERNAL_VERSION) return true;
	if (!value || typeof value !== 'object') return false;
	const obj = value as { id?: unknown; tag?: unknown };
	return typeof obj.id === 'string' && typeof obj.tag === 'string';
}

function isSelectedRunner(value: unknown): value is SelectedRunner {
	if (value === GLOBAL_RUNNER) return true;
	if (!value || typeof value !== 'object') return false;
	const obj = value as { name?: unknown; path?: unknown };
	return typeof obj.name === 'string' && typeof obj.path === 'string';
}

export function isExecutionTarget(value: unknown): value is ExecutionTarget {
	if (!value || typeof value !== 'object') return false;
	const obj = value as {
		wallet?: unknown;
		version?: unknown;
		runner?: unknown;
	};
	return (
		'wallet' in obj &&
		isHubItemLike(obj.wallet) &&
		'version' in obj &&
		isSelectedVersion(obj.version) &&
		'runner' in obj &&
		isSelectedRunner(obj.runner)
	);
}
