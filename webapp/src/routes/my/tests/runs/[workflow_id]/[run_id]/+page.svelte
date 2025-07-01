<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script>
	import { page } from '$app/state';
	import BackButton from '$lib/layout/back-button.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	//

	let { data } = $props();
	const { workflowId, runId } = data;
</script>

<div class="bg-primary text-white">
	<div class="!px-2 md:!px-4 lg:!px-8">
		<BackButton href="/my/tests/runs" class="text-white">{m.Back_to_test_runs()}</BackButton>
	</div>
</div>

<PageTop contentClass="!space-y-0 !px-2 md:!px-4 lg:!px-8">
	<T tag="h2">{m.Test_run()}: {workflowId}</T>
</PageTop>

<iframe title="Workflow" src={page.url.pathname + '/temporal'} class="h-screen w-full"></iframe>
