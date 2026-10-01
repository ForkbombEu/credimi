// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { joinPipelineYamlPreview, splitPipelineYamlPreview } from './yaml-preview-split.js';

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

describe('splitPipelineYamlPreview', () => {
	it('returns empty parts for empty input', () => {
		expect(splitPipelineYamlPreview('')).toEqual({
			header: '',
			steps: [],
			followUpsPreamble: '',
			followUps: [],
			betweenFollowUps: []
		});
	});

	it('keeps header-only yaml when there are no step ranges', () => {
		const yaml = 'name: alone\n';
		expect(splitPipelineYamlPreview(yaml)).toEqual({
			header: yaml,
			steps: [],
			followUpsPreamble: '',
			followUps: [],
			betweenFollowUps: []
		});
	});

	it('splits header, steps, and follow-ups index-aligned with cards', () => {
		const parts = splitPipelineYamlPreview(SAMPLE);

		expect(parts.header).toContain('name: demo');
		expect(parts.header).toContain('runtime:');
		expect(parts.header).toMatch(/steps:\s*$/);
		expect(parts.header).not.toContain('email-0001');

		expect(parts.steps).toHaveLength(2);
		expect(parts.steps[0]?.index).toBe(0);
		expect(parts.steps[0]?.text).toContain('email-0001');
		expect(parts.steps[1]?.index).toBe(1);
		expect(parts.steps[1]?.text).toContain('http-0002');

		expect(parts.followUpsPreamble).toContain('finally:');
		expect(parts.followUpsPreamble).toContain('always:');
		expect(parts.followUpsPreamble).not.toContain('email-0003');

		expect(parts.followUps).toHaveLength(2);
		expect(parts.followUps[0]?.index).toBe(0);
		expect(parts.followUps[0]?.text).toContain('email-0003');
		expect(parts.followUps[1]?.index).toBe(1);
		expect(parts.followUps[1]?.text).toContain('http-0004');

		expect(parts.betweenFollowUps).toHaveLength(2);
		expect(parts.betweenFollowUps[0]).toContain('on_success:');
		expect(parts.betweenFollowUps[1]).toBe('');
	});

	it('rejoins to the original document when fragments are concatenated', () => {
		const parts = splitPipelineYamlPreview(SAMPLE);
		// Trailing document newline may be dropped; compare without a final empty line.
		expect(joinPipelineYamlPreview(parts).replace(/\n$/, '')).toBe(SAMPLE.replace(/\n$/, ''));
	});

	it('handles debug steps that start with use', () => {
		const yaml = `name: x\n\nsteps:\n  - use: debug\n\n  - id: email-0001\n    use: email\n`;
		const parts = splitPipelineYamlPreview(yaml);
		expect(parts.steps).toHaveLength(2);
		expect(parts.steps[0]?.text).toContain('use: debug');
		expect(parts.steps[1]?.text).toContain('email-0001');
	});
});
