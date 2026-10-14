// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { parse, stringify, type ScalarTag } from 'yaml';

import type { Pipeline } from './types';

//

export function parseYaml(yaml: string): Pipeline {
	return parse(yaml) as Pipeline;
}

/**
 * Numeric scalars `gopkg.in/yaml.v3` resolves as numbers but that neither the
 * YAML 1.1 nor the YAML 1.2 (`core`) schema of the `yaml` package recognises as
 * one, so the emitter would leave the author's string unquoted:
 *
 * - `0o17` / `0O17`: the backend parses with `strconv.ParseInt(plain, 0, 64)`,
 *   whose base-0 Go syntax accepts the `0o` prefix; YAML 1.1 spells octal `017`
 *   and does not know `0o`.
 * - `0X1F` / `0B101`: the schema regexes only match lower-case prefixes, Go
 *   accepts either case.
 * - `1_000` / `0_x_1f` / `1e1_0`: the backend strips every `_` before parsing,
 *   so separators may sit anywhere between the digits and the prefix, while the
 *   schema regexes only allow them between digits.
 *
 * The emitter consults `test` of every implicit (`default`) tag before writing
 * a plain string scalar, so adding this tag to the schema makes such strings
 * come out quoted. `resolve` mirrors the backend conversion, for the case the
 * schema is used to parse rather than stringify.
 */
const backendNumberTag: ScalarTag = {
	tag: 'tag:yaml.org,2002:int',
	default: true,
	test: /^[-+]?_*(?:0_*[bB][0-1_]+|0_*[oO][0-7_]+|0_*[xX][0-9a-fA-F_]+|[0-9][0-9_]*(?:\.[0-9_]*)?(?:[eE][-+_]*[0-9_]+)?|\.[0-9_]+(?:[eE][-+_]*[0-9_]+)?)$/,
	resolve: (value: string) => {
		const plain = value.replaceAll('_', '');
		const sign = plain.startsWith('-') ? -1 : 1;
		const unsigned = plain.replace(/^[-+]/, '');
		if (/^0[xX]/.test(unsigned)) return sign * Number.parseInt(unsigned.slice(2), 16);
		if (/^0[oO]/.test(unsigned)) return sign * Number.parseInt(unsigned.slice(2), 8);
		if (/^0[bB]/.test(unsigned)) return sign * Number.parseInt(unsigned.slice(2), 2);
		if (/^0[0-7]+$/.test(unsigned)) return sign * Number.parseInt(unsigned.slice(1), 8);
		return sign * Number(unsigned);
	}
};

/**
 * Serialize a pipeline document to YAML safe for the Go consumer.
 *
 * The backend parses pipeline YAML with `gopkg.in/yaml.v3`, which resolves a
 * plain date-like scalar as `!!timestamp` and re-encodes it as RFC3339 in HTTP
 * bodies (`2012-03-04` -> `2012-03-04T00:00:00Z`), and a plain numeric, boolean
 * or null scalar as that type. The `yaml-1.1` schema knows the timestamp type
 * and the YAML 1.1 forms (dates, yes/no/on/off, sexagesimal and `017`-style
 * numbers, underscore separators), so those strings are quoted and survive
 * verbatim; `backendNumberTag` adds the numeric forms resolved only by YAML 1.2
 * (`0o17`) or by the backend's base-0 Go syntax (`0X1F`, `1_000`). Quoting is
 * always safe for a string, while real numbers, booleans and nulls are still
 * emitted plain and stay typed.
 */
export function serializeYaml(value: unknown): string {
	return stringify(value, { schema: 'yaml-1.1', customTags: [backendNumberTag] });
}
