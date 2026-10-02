// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PipelineFinally, PipelineFinallyCondition } from '$lib/pipeline/types';

export function getFinallyConditions(): PipelineFinallyCondition[] {
	return ['always', 'on_success', 'on_failure'];
}

export function getFinallySteps(
	finallyDefinition: PipelineFinally | undefined,
	condition: PipelineFinallyCondition
) {
	if (!finallyDefinition) return [];
	if (Array.isArray(finallyDefinition)) {
		return condition === 'always' ? finallyDefinition : [];
	}
	return finallyDefinition[condition] ?? [];
}
