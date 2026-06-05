<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import type { GenericRecord } from '@/utils/types';

	import Button from '@/components/ui-custom/button.svelte';
	import RenderMD from '@/components/ui-custom/renderMD.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { m } from '@/i18n';
	import { printElement } from '@/utils/printElement';

	type Props = {
		reportUrl: string | undefined;
		sheetTrigger: Snippet<[{ props: GenericRecord; openSheet: () => void }]>;
	};

	let { reportUrl, sheetTrigger }: Props = $props();

	const hasReport = $derived(Boolean(reportUrl));

	const reportPromise = $derived.by(() => {
		if (!reportUrl) return undefined;
		return fetch(reportUrl).then((res) => res.text());
	});

	let reportEl = $state<HTMLDivElement | undefined>();

	function handlePrint() {
		if (!reportEl) return;
		printElement(reportEl);
	}
</script>

{#if hasReport}
	<Sheet>
		{#snippet trigger({ sheetTriggerAttributes: props, openSheet })}
			{@render sheetTrigger({ props, openSheet })}
		{/snippet}
		{#snippet content()}
			{#await reportPromise then report}
				{#if report}
					<div bind:this={reportEl} class="max-w-full min-w-0 p-4">
						<RenderMD
							content={report}
							scrollableTables
							class="prose prose-sm max-w-none prose-headings:text-primary prose-a:text-primary [&_th]:bg-secondary [&_th]:pt-2"
						/>
					</div>
					<div class="absolute right-6 bottom-6">
						<Button onclick={handlePrint}>{m.Print()}</Button>
					</div>
				{/if}
			{/await}
		{/snippet}
	</Sheet>
{/if}
