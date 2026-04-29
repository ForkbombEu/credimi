// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { m } from '@/i18n';

export enum Placeholder {
	PIPELINE_NAME = 'pipeline_name',
	PIPELINE_URL = 'pipeline_url',
	RESULT = 'result',
	PIPELINE_OUTPUT = 'pipeline_output',
	DATE = 'date'
}

export function formatPlaceholder(value: string) {
	return `\${{${value}}}`;
}

const placeholdersDescriptions: Record<Placeholder, string> = {
	[Placeholder.PIPELINE_NAME]: m.placeholder_pipeline_name(),
	[Placeholder.PIPELINE_URL]: m.placeholder_pipeline_url(),
	[Placeholder.RESULT]: m.placeholder_pipeline_result(),
	[Placeholder.PIPELINE_OUTPUT]: m.placeholder_pipeline_output(),
	[Placeholder.DATE]: m.placeholder_pipeline_date()
};

export function getPlaceholderDescription(placeholder: Placeholder) {
	return placeholdersDescriptions[placeholder];
}
