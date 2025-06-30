<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import FocusPageLayout from '$lib/layout/focus-page-layout.svelte';
	import SectionCard from '$lib/layout/section-card.svelte';
	import { m } from '@/i18n';

	let { data } = $props();
</script>

<FocusPageLayout
	title={m.Review_latest_check_runs()}
	backButton={{ href: '/my/tests/runs', title: m.Back_to_check_runs() }}
>
	{#if data.latestCheckRuns}
		{#each data.latestCheckRuns as checkRun}
			<SectionCard title={checkRun.WorkflowId}>
				<div>
					<h1>{checkRun.WorkflowId}</h1>
					<pre>{JSON.stringify(checkRun, null, 2)}</pre>
				</div>
			</SectionCard>
		{/each}
	{/if}
</FocusPageLayout>
