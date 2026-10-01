// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	findRangeForUnit,
	findNearestUnitToLine,
	findUnitAtLine,
	mapYamlCardRanges
} from './yaml-ranges.js';

const SAMPLE = `name: demo

runtime:
  global_timeout: 60

steps:
  - id: email-0001
    use: email
    with:
      payload:
        recipient: a@b.c

  - id: http-0002
    use: http-request
    with:
      payload:
        url: https://example.com

finally:
  always:
    - id: email-0003
      use: email
      with:
        payload:
          recipient: x@y.z
  on_success:
    - id: http-0004
      use: http-request
      with:
        payload:
          url: https://ok.example
`;

describe('mapYamlCardRanges', () => {
	it('maps top-level steps by index', () => {
		const ranges = mapYamlCardRanges(SAMPLE);
		const steps = ranges.filter((r) => r.section === 'steps');
		expect(steps).toHaveLength(2);
		expect(steps[0]?.index).toBe(0);
		expect(steps[1]?.index).toBe(1);
		const first = findRangeForUnit(ranges, 'steps', 0);
		expect(SAMPLE.split('\n')[first!.startLine]).toContain('- id: email-0001');
		expect(findUnitAtLine(ranges, first!.startLine)?.index).toBe(0);
	});

	it('maps finally items as follow-ups in document order', () => {
		const ranges = mapYamlCardRanges(SAMPLE);
		const followUps = ranges.filter((r) => r.section === 'follow-ups');
		expect(followUps).toHaveLength(2);
		expect(followUps[0]?.index).toBe(0);
		expect(followUps[1]?.index).toBe(1);
		const line = SAMPLE.split('\n').findIndex((l) => l.includes('email-0003'));
		expect(findUnitAtLine(ranges, line)?.section).toBe('follow-ups');
	});

	it('handles debug steps that start with use', () => {
		const yaml = `name: x\n\nsteps:\n  - use: debug\n\n  - id: email-0001\n    use: email\n`;
		const ranges = mapYamlCardRanges(yaml);
		expect(ranges.filter((r) => r.section === 'steps')).toHaveLength(2);
	});

	it('treats continue_on_error-first list items as step boundaries', () => {
		const yaml = `name: fcaf-ish

steps:
  - id: onboard-0001
    use: mobile-automation

  - continue_on_error: true
    id: http-0002
    use: http-request
    with:
      method: GET
      url: https://example.com

  - continue_on_error: true
    id: http-0003
    use: http-request
`;
		const steps = mapYamlCardRanges(yaml).filter((r) => r.section === 'steps');
		expect(steps).toHaveLength(3);
		const lines = yaml.split('\n');
		expect(lines[steps[1]!.startLine]).toMatch(/continue_on_error/);
		expect(lines[steps[2]!.startLine]).toMatch(/continue_on_error/);
	});

	it('treats continue_on_error-first finally items as follow-up boundaries', () => {
		const yaml = `name: x

steps:
  - id: a-0001
    use: debug

finally:
  always:
    - continue_on_error: true
      id: email-0002
      use: email
    - id: http-0003
      use: http-request
`;
		const followUps = mapYamlCardRanges(yaml).filter((r) => r.section === 'follow-ups');
		expect(followUps).toHaveLength(2);
		expect(yaml.split('\n')[followUps[0]!.startLine]).toMatch(/continue_on_error/);
	});
});

describe('findNearestUnitToLine', () => {
	it('returns exact hits like findUnitAtLine', () => {
		const ranges = mapYamlCardRanges(SAMPLE);
		const first = findRangeForUnit(ranges, 'steps', 0)!;
		expect(findNearestUnitToLine(ranges, first.startLine)?.index).toBe(0);
	});

	it('maps blank gaps between steps to the nearer range', () => {
		const ranges = mapYamlCardRanges(SAMPLE);
		const a = findRangeForUnit(ranges, 'steps', 0)!;
		const b = findRangeForUnit(ranges, 'steps', 1)!;
		expect(b.startLine - a.endLine).toBeGreaterThan(1);
		const gap = a.endLine + 1;
		expect(findUnitAtLine(ranges, gap)).toBeUndefined();
		const nearest = findNearestUnitToLine(ranges, gap);
		expect(nearest?.section).toBe('steps');
		expect(nearest?.index).toBe(0);
	});

	it('clears above the first step', () => {
		const ranges = mapYamlCardRanges(SAMPLE);
		const start = findRangeForUnit(ranges, 'steps', 0)!.startLine;
		expect(findNearestUnitToLine(ranges, start - 1)).toBeUndefined();
	});
});
