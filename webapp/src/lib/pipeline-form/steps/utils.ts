// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';
import type { PipelineStep, PipelineStepType } from '$lib/pipeline/types';

import BugIcon from '@lucide/svelte/icons/bug';

import { m } from '@/i18n';

import { getConfigByTypeOrThrow } from '.';

//

export function getDisplayData(type: string): EntityData {
	if (type === 'debug') {
		return debugEntityData;
	}
	return getConfigByTypeOrThrow(type as PipelineStepType).display;
}

export function formatLinkedId(step: PipelineStep) {
	if (!('id' in step)) throw new Error(m.Pipeline_form_step_has_no_id());
	let deeplinkPath = '.outputs';
	if (step.use === 'conformance-check') {
		deeplinkPath += '.deeplink';
	}
	return '${{' + step.id + deeplinkPath + '}}';
}

export const debugEntityData: EntityData = {
	slug: 'debug',
	icon: BugIcon,
	labels: {
		singular: m.Debug()
	},
	classes: {
		bg: 'bg-gray-100',
		text: 'text-gray-500',
		border: 'border-gray-500'
	}
};
