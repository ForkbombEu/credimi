// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	findRangeForUnit,
	findUnitAtLine,
	firstStepStartLine,
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

	it('reports first step start for header wash clearing', () => {
		const ranges = mapYamlCardRanges(SAMPLE);
		const start = firstStepStartLine(ranges);
		expect(start).not.toBeNull();
		expect(start!).toBeGreaterThan(0);
		expect(SAMPLE.split('\n')[start!]).toMatch(/email-0001/);
	});

	it('handles debug steps that start with use', () => {
		const yaml = `name: x\n\nsteps:\n  - use: debug\n\n  - id: email-0001\n    use: email\n`;
		const ranges = mapYamlCardRanges(yaml);
		expect(ranges.filter((r) => r.section === 'steps')).toHaveLength(2);
	});
});
