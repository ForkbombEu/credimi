// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';
import { parse } from 'yaml';

import type { PipelinesResponse } from '@/pocketbase/types';

import type { Pipeline } from './types';

//

export function parseYaml(yaml: string): Pipeline {
	return parse(yaml) as Pipeline;
}

export function getManualEditHref(pipeline: PipelinesResponse): string {
	return `/my/pipelines/${getPath(pipeline, true)}/edit/manual`;
}
