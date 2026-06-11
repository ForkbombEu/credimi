<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { onMount } from 'svelte';

	import PipelineReportView from '$lib/pipeline/results/pipeline-report-view.svelte';
	import { endPipelineReportBootLoading } from '$lib/pipeline/results/pipeline-report-page';

	let { data } = $props();

	let report = $state<string | undefined>();

	onMount(() => {
		fetch(data.reportUrl)
			.then((res) => res.text())
			.then((text) => {
				report = text;
			})
			.finally(() => {
				endPipelineReportBootLoading();
			});
	});
</script>

{#if report}
	<PipelineReportView content={report} />
{/if}
