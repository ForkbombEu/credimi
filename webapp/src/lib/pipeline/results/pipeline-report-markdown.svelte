<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import RenderMD from '@/components/ui-custom/renderMD.svelte';

	type Props = {
		content: string;
	};

	let { content }: Props = $props();
</script>

<RenderMD
	{content}
	scrollableTables
	class="prose prose-sm max-w-none prose-headings:text-primary prose-a:text-primary [&_th]:bg-secondary [&_th]:pt-2"
/>
