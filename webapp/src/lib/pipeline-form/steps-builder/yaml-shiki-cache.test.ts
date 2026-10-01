// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('shiki', () => ({
	codeToHtml: vi.fn(async (content: string) => `<pre data-test="${content.length}"></pre>`)
}));

import { codeToHtml } from 'shiki';

import {
	clearYamlShikiCache,
	highlightYamlFragment,
	yamlShikiCacheKey,
	yamlShikiCacheSize,
	YAML_SHIKI_CACHE_MAX
} from './yaml-shiki-cache.js';

describe('yaml-shiki-cache', () => {
	afterEach(() => {
		clearYamlShikiCache();
		vi.mocked(codeToHtml).mockClear();
	});

	it('builds a stable cache key from theme and content', () => {
		expect(yamlShikiCacheKey('a: 1', 'github-light')).toBe('github-light\0a: 1');
	});

	it('highlights once per content/theme and serves cache hits', async () => {
		const a = await highlightYamlFragment('use: debug', 'catppuccin-frappe');
		const b = await highlightYamlFragment('use: debug', 'catppuccin-frappe');

		expect(a).toBe(b);
		expect(codeToHtml).toHaveBeenCalledTimes(1);
		expect(yamlShikiCacheSize()).toBe(1);
	});

	it('re-highlights when content changes', async () => {
		await highlightYamlFragment('a: 1', 'catppuccin-frappe');
		await highlightYamlFragment('a: 2', 'catppuccin-frappe');

		expect(codeToHtml).toHaveBeenCalledTimes(2);
		expect(yamlShikiCacheSize()).toBe(2);
	});

	it('evicts the oldest entry when the soft cap is exceeded', async () => {
		for (let i = 0; i < YAML_SHIKI_CACHE_MAX + 3; i++) {
			await highlightYamlFragment(`k: ${i}`, 'catppuccin-frappe');
		}
		expect(yamlShikiCacheSize()).toBe(YAML_SHIKI_CACHE_MAX);
		// First keys were evicted — re-highlighting them calls Shiki again.
		vi.mocked(codeToHtml).mockClear();
		await highlightYamlFragment('k: 0', 'catppuccin-frappe');
		expect(codeToHtml).toHaveBeenCalledTimes(1);
	});
});
