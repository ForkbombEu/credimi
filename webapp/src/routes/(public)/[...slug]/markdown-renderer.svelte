<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { marked } from 'marked';

	type Props = {
		content: string;
		class?: string;
		preprocessHtml?: (html: string) => string;
	};

	let { content, class: className = '', preprocessHtml }: Props = $props();

	const html = $derived.by(() => {
		const baseHtml = marked(content, { async: false });
		return preprocessHtml ? preprocessHtml(baseHtml) : baseHtml;
	});
</script>

<div class="prose prose-headings:font-serif prose-h1:text-3xl {className}">{@html html}</div>
