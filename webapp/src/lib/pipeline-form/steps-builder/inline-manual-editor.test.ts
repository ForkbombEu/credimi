// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

vi.mock('$lib', async () => {
	const { validateYaml } = await import('$lib/pipeline/validate-yaml');
	return { Pipeline: { validateYaml } };
});

import { InlineManualEditor } from './inline-manual-editor.svelte.js';

const VALID_YAML = `name: test

steps:
  - use: debug
`;

describe('InlineManualEditor', () => {
	it('tracks dirty state', () => {
		const editor = new InlineManualEditor(VALID_YAML);
		expect(editor.isDirty).toBe(false);
		editor.yaml = `${VALID_YAML}\n`;
		expect(editor.isDirty).toBe(true);
		editor.dispose();
	});

	it('validateNow flushes debounce and returns validation', async () => {
		const editor = new InlineManualEditor(VALID_YAML);
		const result = await editor.validateNow();
		expect(result.ok).toBe(true);
		if (result.ok) expect(result.value).toBe(VALID_YAML);
		editor.dispose();
	});

	it('dispose cancels without throwing', () => {
		const editor = new InlineManualEditor(VALID_YAML);
		editor.yaml = 'broken: [';
		expect(() => editor.dispose()).not.toThrow();
	});
});
