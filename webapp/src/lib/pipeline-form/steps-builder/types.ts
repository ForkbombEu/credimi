// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { GenericRecord } from '@/utils/types';

import type { PipelineStep } from '../types';

//

export type EnrichedStep = [PipelineStep, GenericRecord | Enrich404Error | Error];

export class Enrich404Error extends Error {
	constructor() {
		super('Resource not found');
	}
}
