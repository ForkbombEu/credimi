// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineStep } from '$lib/pipeline/types';

/**
 * Stable YAML key order for pipeline step cards:
 * use → id → continue_on_error → activity_options → with
 * (omit absent keys; keep any unexpected keys after these).
 */
const STEP_YAML_KEY_ORDER = ['use', 'id', 'continue_on_error', 'activity_options', 'with'] as const;

export function orderStepKeys(step: PipelineStep): PipelineStep {
	const source = step as Record<string, unknown>;
	const ordered: Record<string, unknown> = {};
	for (const key of STEP_YAML_KEY_ORDER) {
		if (Object.prototype.hasOwnProperty.call(source, key)) {
			ordered[key] = key === 'with' ? orderStepWith(step.use, source[key]) : source[key];
		}
	}
	for (const key of Object.keys(source)) {
		if (!(key in ordered)) {
			ordered[key] = source[key];
		}
	}
	return ordered as PipelineStep;
}

/** Plain object (not null, not array). */
function isPlainObject(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** Preferred leading keys inside `http-request` `with`. */
const HTTP_REQUEST_WITH_KEY_ORDER = ['url', 'method', 'expected_status'] as const;

function orderStepWith(use: PipelineStep['use'], value: unknown): unknown {
	const ordered = orderMapKeysByValueKind(value);
	if (use !== 'http-request' || !isPlainObject(ordered)) {
		return ordered;
	}
	return preferObjectKeys(ordered, HTTP_REQUEST_WITH_KEY_ORDER);
}

/** Move preferred keys to the front (when present); keep remaining keys as-is. */
function preferObjectKeys(
	source: Record<string, unknown>,
	preferred: readonly string[]
): Record<string, unknown> {
	const ordered: Record<string, unknown> = {};
	for (const key of preferred) {
		if (Object.prototype.hasOwnProperty.call(source, key)) {
			ordered[key] = source[key];
		}
	}
	for (const key of Object.keys(source)) {
		if (!(key in ordered)) {
			ordered[key] = source[key];
		}
	}
	return ordered;
}

/**
 * Recursively reorder object keys: scalars, then arrays, then nested objects.
 * Nested objects are ordered by ascending immediate key count (ties keep input order).
 */
export function orderMapKeysByValueKind(value: unknown): unknown {
	if (Array.isArray(value)) {
		return value.map(orderMapKeysByValueKind);
	}
	if (!isPlainObject(value)) {
		return value;
	}

	const entries = Object.entries(value).map(
		([key, child]) => [key, orderMapKeysByValueKind(child)] as const
	);
	const scalars = entries.filter(([, child]) => !isPlainObject(child) && !Array.isArray(child));
	const arrays = entries.filter(([, child]) => Array.isArray(child));
	const objects = entries
		.filter(([, child]) => isPlainObject(child))
		.sort(
			([, a], [, b]) =>
				Object.keys(a as Record<string, unknown>).length -
				Object.keys(b as Record<string, unknown>).length
		);

	const ordered: Record<string, unknown> = {};
	for (const [key, child] of [...scalars, ...arrays, ...objects]) {
		ordered[key] = child;
	}
	return ordered;
}
