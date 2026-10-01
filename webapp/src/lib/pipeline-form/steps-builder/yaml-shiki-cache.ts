// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { codeToHtml, type BundledTheme } from 'shiki';

/** Soft cap so idle composer sessions do not retain unbounded Shiki HTML. */
export const YAML_SHIKI_CACHE_MAX = 256;

const cache = new Map<string, string>();

export function yamlShikiCacheKey(content: string, theme: BundledTheme): string {
	return `${theme}\0${content}`;
}

export function clearYamlShikiCache(): void {
	cache.clear();
}

export function yamlShikiCacheSize(): number {
	return cache.size;
}

/**
 * Highlight a YAML fragment with Shiki, caching by theme + exact content.
 * Only dirty fragments miss; mounted virtual blocks call this independently.
 */
export async function highlightYamlFragment(content: string, theme: BundledTheme): Promise<string> {
	const key = yamlShikiCacheKey(content, theme);
	const hit = cache.get(key);
	if (hit !== undefined) return hit;

	const html = await codeToHtml(content, {
		lang: 'yaml',
		theme,
		transformers: [
			{
				pre(node) {
					this.addClassToHast(node, [
						'yaml-preview-block-pre',
						'w-full',
						'overflow-x-auto',
						'text-sm',
						'm-0',
						'rounded-none',
						'border-0'
					]);
				},
				line(node) {
					this.addClassToHast(node, 'code-display-line');
					// Empty source lines are empty spans; NBSP keeps blank rows one line tall.
					if (!node.children || node.children.length === 0) {
						node.children = [{ type: 'text', value: '\u00A0' }];
					}
				}
			}
		]
	});

	if (cache.size >= YAML_SHIKI_CACHE_MAX) {
		const oldest = cache.keys().next().value;
		if (oldest !== undefined) cache.delete(oldest);
	}
	cache.set(key, html);
	return html;
}
