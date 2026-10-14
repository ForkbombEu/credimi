// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';
import { isScalar, parse, parseDocument } from 'yaml';

import { serializeYaml } from './utils';

// The backend resolves pipeline YAML with gopkg.in/yaml.v3, so a value survives
// only if neither schema re-reads it as another type: YAML 1.1 knows timestamps,
// yes/no/on/off, sexagesimal and `017`-octal scalars, YAML 1.2 (`core`) knows
// `0o17`-octal, and yaml.v3 itself also accepts `0X1F`/`0B101` with either case
// and ignores `_` separators anywhere (base-0 Go literals).
const schemas = ['yaml-1.1', 'core'] as const;

const reread = (yaml: string) => schemas.map((schema) => parse(yaml, { schema }));

const scalarStyle = (yaml: string) => {
	const node = parseDocument(yaml).getIn(['value'], true);
	return isScalar(node) ? node.type : undefined;
};

describe('serializeYaml', () => {
	// A string the backend resolves to a number, boolean, null or timestamp must
	// come out quoted, otherwise the author's text is retyped (`0o17` -> 15) or
	// rewritten (`2012-03-04` -> `2012-03-04T00:00:00Z`) on the way to the activity.
	const ambiguousStrings = [
		// yaml.v3 timestamp layouts
		'2012-03-04',
		'2012-3-4',
		'2012-03-04 12:00:00',
		'2031-01-01T00:00:00Z',
		'2025-10-14T00:00:00.000Z',
		'2001-12-14t21:59:43.10-05:00',
		// integers: decimal, 1.1/1.2 octals, hex, binary, with case and underscore variants
		'0',
		'-1',
		'+1',
		'123',
		'1_000',
		'017',
		'0o17',
		'0O17',
		'0o1_7',
		'0_x_1f',
		'0x1f',
		'0X1F',
		'0b101',
		'0B101',
		// floats
		'1.5',
		'.5',
		'1.',
		'1e3',
		'1E3',
		'1e+3',
		'1_000.5',
		'1e1_0',
		'.inf',
		'-.INF',
		'.NaN',
		// YAML 1.1 booleans, nulls and sexagesimal numbers
		'true',
		'True',
		'false',
		'FALSE',
		'yes',
		'no',
		'on',
		'off',
		'y',
		'N',
		'null',
		'~',
		'1:30',
		'12:00:00'
	];

	it.each(ambiguousStrings)('quotes the string %j for the backend parser', (value) => {
		const out = serializeYaml({ value });

		expect(scalarStyle(out)).not.toBe('PLAIN');
		for (const doc of reread(out)) expect(doc).toEqual({ value });
	});

	it.each([
		{ int: 15, negative: -3, float: 1.5, exponent: 1e3 },
		{ yes: true, no: false },
		{ nothing: null }
	])('leaves %j typed', (value) => {
		const out = serializeYaml(value);

		for (const doc of reread(out)) expect(doc).toEqual(value);
	});
});
