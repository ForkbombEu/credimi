// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { parse, stringify } from 'yaml';

import type { Pipeline } from './types';

//

export function parseYaml(yaml: string): Pipeline {
	return parse(yaml) as Pipeline;
}

/**
 * Serialize a pipeline document to YAML safe for the Go consumer.
 *
 * The backend parses pipeline YAML with `gopkg.in/yaml.v3`, which resolves a
 * plain date-like scalar as `!!timestamp` and re-encodes it as RFC3339 in
 * HTTP bodies (`2012-03-04` -> `2012-03-04T00:00:00Z`). The default `core`
 * schema treats such strings as plain scalars and emits them unquoted, which
 * silently rewrites the author's value. The `yaml-1.1` schema knows the
 * timestamp type, so date-like strings are quoted and survive verbatim.
 */
export function serializeYaml(value: unknown): string {
	return stringify(value, { schema: 'yaml-1.1' });
}
