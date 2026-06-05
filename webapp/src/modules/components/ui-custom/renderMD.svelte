<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { parseMarkdown } from '@/components/ui-custom/markdown-renderer';
	import HTML from '@/components/ui-custom/renderHTML.svelte';

	type Props = {
		content: string;
		class?: string;
		scrollableTables?: boolean;
	};

	const { content, class: className, scrollableTables = false }: Props = $props();

	const html = $derived(
		parseMarkdown(content, {
			table: { scroll: scrollableTables }
		})
	);
</script>

<HTML class={className} content={html} />
