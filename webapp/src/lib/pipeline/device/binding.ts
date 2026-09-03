// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { lsSync } from 'rune-sync/localstorage';

import type { PipelinesResponse } from '@/pocketbase/types';

import type { DeviceRecord } from './types';

import { parseYaml } from '../utils';

//

export function isRequired(p: PipelinesResponse): boolean {
	const yaml = parseYaml(p.yaml);
	return (yaml?.steps ?? []).some((step) => step.use === 'mobile-automation');
}

export function getType(pipeline: PipelinesResponse): 'global' | 'specific' | 'not-needed' {
	const yaml = parseYaml(pipeline.yaml);
	const steps = (yaml?.steps ?? []).filter((step) => step.use === 'mobile-automation');

	if (steps.length === 0) return 'not-needed';

	const areAllStepsSpecific = steps.every((step) => step.with.device_id);
	if (areAllStepsSpecific) return 'specific';

	const areSomeStepsSpecific = steps.some((step) => step.with.device_id);
	if (areSomeStepsSpecific) throw new Error('Mixed device types');

	return 'global';
}

export function getExecutionDevicePath(pipeline: PipelinesResponse): string | undefined {
	const type = getType(pipeline);
	if (type === 'not-needed') return undefined;
	if (type === 'global') return get(pipeline.id);
	if (type === 'specific') {
		const yaml = parseYaml(pipeline.yaml);
		const step = (yaml?.steps ?? []).find((s) => s.use === 'mobile-automation');
		const deviceId = step && 'with' in step ? step.with?.device_id : undefined;
		return typeof deviceId === 'string' ? deviceId : undefined;
	}
	return undefined;
}

type PipelinesDevicesConfig = Record<string, string>;

const pipelinesDevicesConfig = lsSync<PipelinesDevicesConfig>('pipelines_devices_config', {});

export function set(pipeline: PipelinesResponse, device: Pick<DeviceRecord, 'path'>): void {
	try {
		pipelinesDevicesConfig[pipeline.id] = device.path;
	} catch (error) {
		console.error('Failed to set pipeline device:', error);
	}
}

export function get(pipelineId: string): string | undefined {
	try {
		return pipelinesDevicesConfig[pipelineId];
	} catch (error) {
		console.error('Failed to get pipeline device:', error);
		return undefined;
	}
}
