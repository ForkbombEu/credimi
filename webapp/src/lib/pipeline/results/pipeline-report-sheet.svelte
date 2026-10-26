<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { ExternalLinkIcon } from '@lucide/svelte';
	import { buildPipelineReportPageHref } from '$lib/pipeline/results/pipeline-report-page';

	import type { GenericRecord } from '@/utils/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { m } from '@/i18n';

	type Props = {
		reportUrl: string | undefined;
		sheetTrigger: Snippet<[{ props: GenericRecord; openSheet: () => void }]>;
	};

	let { reportUrl, sheetTrigger }: Props = $props();

	const hasReport = $derived(Boolean(reportUrl));

	const reportPageHref = $derived(reportUrl ? buildPipelineReportPageHref(reportUrl) : undefined);
</script>

{#if hasReport}
	<Sheet>
		{#snippet trigger({ sheetTriggerAttributes: props, openSheet })}
			{@render sheetTrigger({ props, openSheet })}
		{/snippet}
		{#snippet content()}
			<!-- Fetched once the sheet opens: the report and its markdown view are only needed there. -->
			{#await Promise.all( [fetch(reportUrl!).then( (res) => res.text() ), import('./pipeline-report-view.svelte')] ) then [report, { default: PipelineReportView }]}
				{#if report}
					<PipelineReportView
						content={report}
						contentClass="max-w-full min-w-0 p-4"
						actionsClass="absolute right-6 bottom-6 flex gap-2"
					>
						{#snippet actions()}
							{#if reportPageHref}
								<Button variant="outline" href={reportPageHref} target="_blank">
									<ExternalLinkIcon class="size-4" />
									{m.Open_in_new_page()}
								</Button>
							{/if}
						{/snippet}
					</PipelineReportView>
				{/if}
			{/await}
		{/snippet}
	</Sheet>
{/if}
