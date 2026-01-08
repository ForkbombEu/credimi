// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';

import { configs } from '$lib/pipeline-form/steps';
import BugIcon from 'lucide-svelte/icons/bug';

import { m } from '@/i18n';

import type { EnrichedStep } from '../steps-builder.svelte.js';

//

export function getStepDisplayData(type: string): EntityData {
	if (type === 'debug') {
		return debugEntityData;
	} else {
		const config = configs.find((c) => c.id === type);
		if (!config) throw new Error(`Unknown step type: ${type}`);
		return config.display;
	}
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

export function getStepCardData(step: EnrichedStep): {
	title: string;
	copyText?: string;
	avatar?: string;
} {
	if (step[0].use === 'debug') {
		return { title: m.Debug() };
	} else {
		const config = configs.find((c) => c.id === step[0].use);
		if (!config) throw new Error(`Unknown step type: ${step[0].use}`);
		return config.cardData(step[1]);
	}
}
