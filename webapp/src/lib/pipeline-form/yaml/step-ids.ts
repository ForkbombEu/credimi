// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStep } from '$lib/pipeline/types';

import * as _ from 'lodash';
import slugify from 'slugify';

import { getConfigByTypeOrThrow } from '../steps';

export function seedIdCounters(steps: PipelineStep[]) {
	const counters = new Map<string, number>();
	for (const step of steps) {
		const base = getIdBase(step);
		if (!base || !('id' in step)) continue;
		const suffix = getIdSuffix(step.id, base);
		if (suffix === undefined) continue;
		counters.set(base, Math.max(counters.get(base) ?? 0, suffix));
	}
	return counters;
}

export function assignStepId(step: PipelineStep, counters: Map<string, number>) {
	const base = getIdBase(step);
	if (!base || !('id' in step)) return;

	const existingSuffix = getIdSuffix(step.id, base);
	if (existingSuffix !== undefined) {
		counters.set(base, Math.max(counters.get(base) ?? 0, existingSuffix));
		return;
	}

	const nextSuffix = (counters.get(base) ?? 0) + 1;
	counters.set(base, nextSuffix);
	step.id = `${base}-${nextSuffix.toString().padStart(4, '0')}`;
}

function getIdBase(step: PipelineStep): string | undefined {
	if (step.use === 'debug' || !('with' in step)) return undefined;
	try {
		const config = getConfigByTypeOrThrow(step.use);
		return slugify(config.makeId(step.with));
	} catch {
		// One bad makeId (e.g. template URL) must not blank the whole YAML preview.
		return slugify(step.use);
	}
}

function getIdSuffix(id: string, base: string) {
	const match = id.match(new RegExp(`^${_.escapeRegExp(base)}-(\\d+)$`));
	return match ? Number(match[1]) : undefined;
}
