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

	it('asks Shiki for a transparent pre background via transformers', async () => {
		await highlightYamlFragment('a: 1', 'catppuccin-frappe');
		expect(codeToHtml).toHaveBeenCalledWith(
			'a: 1',
			expect.objectContaining({
				lang: 'yaml',
				theme: 'catppuccin-frappe',
				transformers: expect.any(Array)
			})
		);
		const opts = vi.mocked(codeToHtml).mock.calls[0]?.[1] as {
			transformers: Array<{ pre?: (node: { properties: Record<string, unknown> }) => void }>;
		};
		const pre = opts.transformers[0]?.pre;
		expect(pre).toBeTypeOf('function');
		const node = {
			properties: {
				style: 'background-color:#303446;color:#c6d0f5',
				class: 'shiki'
			}
		};
		pre!.call(
			{
				addClassToHast(n: { properties: Record<string, unknown> }, classes: string[]) {
					const prev = typeof n.properties.class === 'string' ? n.properties.class : '';
					n.properties.class = `${prev} ${classes.join(' ')}`.trim();
				}
			},
			node
		);
		expect(String(node.properties.style)).not.toMatch(/background-color/i);
		expect(String(node.properties.class)).toContain('bg-transparent');
		expect(String(node.properties.class)).toContain('scrollbar-on-dark');
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
