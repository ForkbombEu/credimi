<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import type { GenericRecord } from '@/utils/types';

	import Sheet from '@/components/ui-custom/sheet.svelte';

	import ReportView from './report-view.svelte';
	import { loadReport } from './report.js';

	type Props = {
		reportUrl: string | undefined;
		pdfUrl: string | undefined;
		sheetTrigger: Snippet<[{ props: GenericRecord; openSheet: () => void }]>;
	};

	let { reportUrl, pdfUrl, sheetTrigger }: Props = $props();
</script>

{#if reportUrl}
	<Sheet title="FCAF assessment" class="sm:max-w-3xl">
		{#snippet trigger({ sheetTriggerAttributes: props, openSheet })}
			{@render sheetTrigger({ props, openSheet })}
		{/snippet}
		{#snippet content()}
			{#await loadReport(reportUrl) then report}
				{#if report}
					<ReportView {report} {reportUrl} {pdfUrl} />
				{:else}
					<p class="py-8 text-sm text-muted-foreground">
						Unable to load the FCAF assessment.
					</p>
				{/if}
			{/await}
		{/snippet}
	</Sheet>
{/if}
