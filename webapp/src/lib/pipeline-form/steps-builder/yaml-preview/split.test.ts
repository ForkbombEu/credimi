// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { joinPipelineYamlPreview, splitPipelineYamlPreview } from './index.js';

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

	it('keeps the blank line between continue_on_error-first steps in the prior fragment', () => {
		const yaml = `name: x

steps:
  - continue_on_error: true
    id: a
    use: http-request

  - continue_on_error: true
    id: b
    use: http-request
`;
		const parts = splitPipelineYamlPreview(yaml);
		expect(parts.steps).toHaveLength(2);
		// Inter-step blank stays on the previous fragment so virtual rows keep a true YAML gap.
		expect(parts.steps[0]?.text.endsWith('\n')).toBe(true);
		expect(parts.steps[0]?.text).toMatch(/use: http-request\n$/);
		expect(parts.steps[1]?.text.startsWith('  - continue_on_error:')).toBe(true);
		expect(joinPipelineYamlPreview(parts).replace(/\n$/, '')).toBe(yaml.replace(/\n$/, ''));
	});

	it('treats continue_on_error-first list items as step boundaries (FCAF-ish)', () => {
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
		const parts = splitPipelineYamlPreview(yaml);
		expect(parts.steps).toHaveLength(3);
		expect(parts.steps[0]?.text).toContain('onboard-0001');
		expect(parts.steps[1]?.text).toMatch(/continue_on_error[\s\S]*http-0002/);
		expect(parts.steps[2]?.text).toMatch(/continue_on_error[\s\S]*http-0003/);
		expect(joinPipelineYamlPreview(parts).replace(/\n$/, '')).toBe(yaml.replace(/\n$/, ''));
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
		const parts = splitPipelineYamlPreview(yaml);
		expect(parts.followUps).toHaveLength(2);
		expect(parts.followUps[0]?.text).toMatch(/continue_on_error[\s\S]*email-0002/);
		expect(parts.followUps[1]?.text).toContain('http-0003');
		expect(joinPipelineYamlPreview(parts).replace(/\n$/, '')).toBe(yaml.replace(/\n$/, ''));
	});
});
