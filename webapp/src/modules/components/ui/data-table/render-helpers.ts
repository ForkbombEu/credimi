// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Snippet } from 'svelte';

/**
 * App-wide deferred component payload. Prefer {@link Comp} / {@link comp}
 * from `$lib/renderable`. These aliases keep the data-table adapter stable.
 */
export { Comp as RenderComponentConfig, comp as renderComponent } from '$lib/renderable/comp.js';

/**
 * A helper class to make it easy to identify Svelte Snippets in `columnDef.cell` and `columnDef.header` properties.
 *
 * > NOTE: This class should only be used internally by the adapter. If you're
 * reading this and you don't know what this is for, you probably don't need it.
 *
 * @example
 * ```svelte
 * {@const result = content(context as any)}
 * {#if result instanceof RenderSnippetConfig}
 *   {@const { snippet, params } = result}
 *   {@render snippet(params)}
 * {/if}
 * ```
 */
export class RenderSnippetConfig<TProps> {
	snippet: Snippet<[TProps]>;
	params: TProps;
	constructor(snippet: Snippet<[TProps]>, params: TProps) {
		this.snippet = snippet;
		this.params = params;
	}
}

/**
 * A helper function to help create cells from Svelte Snippets through ColumnDef's `cell` and `header` properties.
 *
 * The snippet must only take one parameter.
 *
 * This is only to be used with Snippets - use `renderComponent` for Svelte Components.
 *
 * @param snippet
 * @param params
 * @returns - A `RenderSnippetConfig` object that helps svelte-table know how to render the header/cell snippet.
 * @example
 * ```ts
 * // +page.svelte
 * const defaultColumns = [
 *   columnHelper.accessor('name', {
 *     cell: cell => renderSnippet(nameSnippet, { name: cell.row.name }),
 *   }),
 *   columnHelper.accessor('state', {
 *     cell: cell => renderSnippet(stateSnippet, { state: cell.row.state }),
 *   }),
 * ]
 * ```
 * @see {@link https://tanstack.com/table/latest/docs/guide/column-defs}
 */
export function renderSnippet<TProps>(snippet: Snippet<[TProps]>, params: TProps = {} as TProps) {
	return new RenderSnippetConfig(snippet, params);
}
