// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { parse } from 'yaml';

import type { Pipeline } from './types';

//

export function parseYaml(yaml: string): Pipeline {
	return parse(yaml) as Pipeline;
}
