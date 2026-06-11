<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { PrinterIcon } from '@lucide/svelte';
	import PipelineReportMarkdown from '$lib/pipeline/results/pipeline-report-markdown.svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { cn } from '@/components/ui/utils.js';
	import { m } from '@/i18n';
	import { printElement } from '@/utils/printElement';

	type Props = {
		content: string;
		contentClass?: string;
		actionsClass?: string;
		actions?: Snippet;
	};

	let {
		content,
		contentClass = 'mx-auto max-w-7xl p-4',
		actionsClass = 'fixed right-6 bottom-6 flex gap-2',
		actions
	}: Props = $props();

	let reportEl = $state<HTMLDivElement | undefined>();

	function handlePrint() {
		if (!reportEl) return;
		printElement(reportEl);
	}
</script>

<div bind:this={reportEl} class={contentClass}>
	<PipelineReportMarkdown {content} />
</div>
<div class={cn(actionsClass)}>
	{@render actions?.()}
	<Button onclick={handlePrint} variant="outline">
		<PrinterIcon class="size-4" />
		{m.Print()}
	</Button>
</div>
