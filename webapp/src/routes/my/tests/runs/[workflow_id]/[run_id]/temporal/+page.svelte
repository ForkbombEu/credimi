<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!-- 
	! IMPORTANT !
	This page has `@` inside its name in order to "reset" the layout stack.
	Styles imported from the `/temporal.css` file conflict with the global styles.
-->

<script lang="ts" module>
	import { z } from 'zod';

	export const HeightMessageSchema = z.object({
		type: z.literal('height'),
		height: z.number()
	});

	function sendHeight(height: number) {
		parent.postMessage({ type: 'height', height }, '/');
	}
</script>

<script lang="ts">
	import { TemporalI18nProvider } from '$lib/temporal';
	import TemporalWorkflow from './temporal-workflow.svelte';
	import { browser } from '$app/environment';
	import { onDestroy } from 'svelte';
	import type { WorkflowResponse } from '../+layout';
	import { WorkflowMessageSchema } from '../+page.svelte';
	import type { HistoryEvent } from '@forkbombeu/temporal-ui';

	//

	let workflow = $state<WorkflowResponse>();
	let eventHistory = $state<HistoryEvent[]>();

	function onMessage(event: MessageEvent) {
		const message = WorkflowMessageSchema.safeParse(event.data);
		if (!message.success) return;

		workflow = message.data.workflow;
		eventHistory = message.data.eventHistory;
	}

	if (browser) {
		window.addEventListener('message', onMessage);

		onDestroy(() => {
			window.removeEventListener('message', onMessage);
		});
	}
</script>

<!--  -->

<!--
	! IMPORTANT !
	temporal.css is exported from @forkbombeu/temporal-ui
	but must be placed in static assets to work properly
-->
<svelte:head>
	<link rel="stylesheet" href="/temporal.css" />
</svelte:head>

<div
	id="temporal-workflow-container"
	class="block"
	bind:offsetHeight={null, (data) => sendHeight(data)}
>
	{#if workflow && eventHistory}
		<TemporalI18nProvider>
			<TemporalWorkflow workflowResponse={workflow} {eventHistory} />
		</TemporalI18nProvider>
	{/if}
</div>
