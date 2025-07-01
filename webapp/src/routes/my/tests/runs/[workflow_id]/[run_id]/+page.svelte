<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import BackButton from '$lib/layout/back-button.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';
	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { onDestroy } from 'svelte';
	import { HeightMessageSchema, WRAPPER_ID } from './temporal/+page.svelte';

	//

	let { data } = $props();
	const { workflowId } = data;

	/* Loading handling and height calculation */

	const iframeId = 'iframe';

	let loading = $state(true);

	function onMessage(event: MessageEvent) {
		const message = HeightMessageSchema.safeParse(event.data);
		if (!message.success) return;

		const iframe = document.getElementById(iframeId);
		if (!iframe || !(iframe instanceof HTMLIFrameElement)) return;

		iframe.height = '';
		iframe.height = message.data.height + 'px';

		loading = false;
	}

	if (browser) {
		window.addEventListener('message', onMessage);

		onDestroy(() => {
			window.removeEventListener('message', onMessage);
		});
	}
</script>

<div class="min-h-screen">
	<div class="bg-primary text-white">
		<div class="!px-2 md:!px-4 lg:!px-8">
			<BackButton href="/my/tests/runs" class="text-white">{m.Back_to_test_runs()}</BackButton
			>
		</div>
	</div>

	<PageTop contentClass="!space-y-0 !px-2 md:!px-4 lg:!px-8">
		<T tag="h2">{m.Test_run()}: {workflowId}</T>
	</PageTop>

	<LoadingDialog {loading} />

	<iframe id={iframeId} title="Workflow" src={page.url.pathname + '/temporal'} class="w-full"
	></iframe>
</div>
