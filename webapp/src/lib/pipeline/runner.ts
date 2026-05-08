// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';
import { lsSync } from 'rune-sync/localstorage';

import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';

import { parseYaml } from './utils';

//

export function isRequired(p: PipelinesResponse): boolean {
	const yaml = parseYaml(p.yaml);
	return (yaml?.steps ?? []).some((step) => step.use === 'mobile-automation');
}

//

export function getType(pipeline: PipelinesResponse): 'global' | 'specific' | 'not-needed' {
	const yaml = parseYaml(pipeline.yaml);
	const steps = (yaml?.steps ?? []).filter((step) => step.use === 'mobile-automation');

	if (steps.length === 0) return 'not-needed';

	const areAllStepsSpecific = steps.every((step) => step.with.runner_id);
	if (areAllStepsSpecific) return 'specific';

	const areSomeStepsSpecific = steps.some((step) => step.with.runner_id);
	if (areSomeStepsSpecific) throw new Error('Mixed runner types');

	return 'global';
}

// Configuration storage

type PipelinesRunnersConfig = Record<string, string>;

const pipelinesRunnersConfig = lsSync<PipelinesRunnersConfig>('pipelines_runners_config', {});

export function set(pipeline: PipelinesResponse, runner: MobileRunnersResponse): void {
	try {
		pipelinesRunnersConfig[pipeline.id] = getPath(runner);
	} catch (error) {
		console.error('Failed to set pipeline runner:', error);
	}
}

export function get(pipelineId: string): string | undefined {
	try {
		return pipelinesRunnersConfig[pipelineId];
	} catch (error) {
		console.error('Failed to get pipeline runner:', error);
		return undefined;
	}
}
