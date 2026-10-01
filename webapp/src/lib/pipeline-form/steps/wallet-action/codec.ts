// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Record } from '$lib/pipeline/device';

import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
import { getHubItemById } from '$lib/hub';
import { resolveByPath } from '$lib/pipeline/device/query.js';
import { type PipelineStepByType, type PipelineStepData } from '$lib/pipeline/types.js';
import { getPath } from '$lib/utils';
import {
	EXTERNAL_VERSION,
	GLOBAL_DEVICE,
	type SelectedDevice,
	type SelectedVersion
} from '$pipeline-form/execution-target/types.js';
import { getLastPathSegment } from '$pipeline-form/steps/_partials/index.js';
import { isError } from 'effect/Predicate';

import { m } from '@/i18n/index.js';
import { type WalletActionsResponse, type WalletVersionsResponse } from '@/pocketbase/types';

import type {
	InlineWalletActionStepData,
	StoredWalletActionStepData,
	WalletActionStepData
} from './types.js';

type MobileWith = PipelineStepData<PipelineStepByType<'mobile-automation'>>;

/**
 * YAML-shape classifier for mobile-automation payloads.
 * Prefers inline when `action_code` is a string and `version_id` is present
 * (same rule as deserialize; stricter than the former makeId check that
 * only required `action_code`).
 */
export function isInlineWalletActionYaml(data: MobileWith): boolean {
	return 'action_code' in data && typeof data.action_code === 'string' && 'version_id' in data;
}

async function resolveVersion(versionId: string): Promise<SelectedVersion> {
	if (versionId === EXTERNAL_VERSION) return EXTERNAL_VERSION;
	const response = await getRecordByCanonifiedPath<WalletVersionsResponse>(versionId);
	if (!isError(response)) return response;
	throw response;
}

function fallbackDevice(path: string): Record {
	return {
		name: getLastPathSegment(path),
		path,
		isOwned: false,
		isPublished: false,
		isOnline: false
	};
}

async function resolveDevice(deviceId: string | undefined): Promise<SelectedDevice> {
	if (!deviceId || deviceId === GLOBAL_DEVICE) return GLOBAL_DEVICE;

	const path = deviceId;
	try {
		return (await resolveByPath(path)) ?? fallbackDevice(path);
	} catch {
		return fallbackDevice(path);
	}
}

function parametersFrom(data: MobileWith): { [key: string]: string } | undefined {
	if (!('parameters' in data) || !data.parameters) return undefined;
	return data.parameters as { [key: string]: string };
}

function versionIdOf(version: SelectedVersion): string {
	return version === EXTERNAL_VERSION ? EXTERNAL_VERSION : getPath(version);
}

/** Shared write of version_id / device_id / parameters onto MobileWith. */
function applyTargetFields(
	_with: MobileWith,
	version: SelectedVersion,
	device: SelectedDevice,
	parameters?: { [key: string]: string }
): void {
	_with.version_id = versionIdOf(version);
	if (device !== GLOBAL_DEVICE) {
		_with.device_id = device.path;
	}
	if (parameters && Object.keys(parameters).length > 0) {
		_with.parameters = parameters;
	}
}

/** Shared resolve of version + device + parameters from YAML. */
async function resolveTargetFields(data: MobileWith): Promise<{
	version: SelectedVersion;
	device: SelectedDevice;
	parameters: { [key: string]: string } | undefined;
}> {
	const version = await resolveVersion(String(data.version_id));
	const device = await resolveDevice(
		'device_id' in data ? (data.device_id as string | undefined) : undefined
	);
	return { version, device, parameters: parametersFrom(data) };
}

export function makeWalletActionId(data: MobileWith): string {
	if (isInlineWalletActionYaml(data)) {
		return 'inline-maestro';
	}
	if (!('action_id' in data) || !('version_id' in data)) {
		throw new Error(m.Pipeline_form_invalid_step_data());
	}
	return getLastPathSegment(data.action_id);
}

export function serializeWalletAction(data: WalletActionStepData): MobileWith {
	if (data.kind === 'inline') {
		const _with = { action_code: data.actionCode } as MobileWith;
		applyTargetFields(_with, data.version, data.device, data.parameters);
		return _with;
	}

	const { action, version, device, parameters } = data;
	const _with = { action_id: getPath(action) } as MobileWith;
	applyTargetFields(_with, version, device, parameters);
	if (!_with.parameters && (action.code.includes('${DL}') || action.code.includes('${deeplink}'))) {
		_with.parameters = {
			deeplink: '<deeplink-placeholder>'
		};
	}
	return _with;
}

export async function deserializeWalletAction(data: MobileWith): Promise<WalletActionStepData> {
	if (isInlineWalletActionYaml(data)) {
		const { version, device, parameters } = await resolveTargetFields(data);
		return {
			kind: 'inline',
			actionCode: data.action_code as string,
			version,
			device,
			parameters
		} satisfies InlineWalletActionStepData;
	}

	if (!('action_id' in data) || !('version_id' in data)) {
		throw new Error(m.Pipeline_form_invalid_step_data());
	}

	const action = await getRecordByCanonifiedPath<WalletActionsResponse>(data.action_id);
	if (isError(action)) {
		throw action;
	}

	const { version, device, parameters } = await resolveTargetFields(data);
	const wallet = await getHubItemById(action.wallet);

	return {
		kind: 'stored',
		wallet,
		version,
		action,
		device,
		parameters
	} satisfies StoredWalletActionStepData;
}
