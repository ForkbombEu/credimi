<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { endPipelineReportBootLoading } from '$lib/pipeline/results/pipeline-report-page';
	import PipelineReportView from '$lib/pipeline/results/pipeline-report-view.svelte';
	import { onMount } from 'svelte';

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
