/*
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
*/

import { marked, type Renderer } from 'marked';

export type MarkdownTableOptions = {
	/** Wrap tables in a scroll container with enhanced presentation styles. */
	scroll?: boolean;
};

export function createMarkdownRenderer(table: MarkdownTableOptions = {}): Renderer {
	if (!table.scroll) {
		return new marked.Renderer();
	}

	const renderer = new marked.Renderer();
	const renderTable = renderer.table.bind(renderer);
	renderer.table = function (token) {
		return `<div class="max-w-full overflow-x-auto overflow-y-clip [&_table]:my-0!">${renderTable(token)}</div>`;
	};

	return renderer;
}

export function parseMarkdown(
	content: string,
	options: { table?: MarkdownTableOptions } = {}
): string {
	return marked.parse(content, {
		async: false,
		renderer: createMarkdownRenderer(options.table)
	});
}
