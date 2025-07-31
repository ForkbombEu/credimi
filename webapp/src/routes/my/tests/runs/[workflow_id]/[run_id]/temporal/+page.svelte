<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!-- 
	! IMPORTANT !
	This page has `@` inside its name in order to "reset" the layout stack.
	Styles imported from the `/temporal.css` file conflict with the global styles.
-->

<script lang="ts">
	import { TemporalI18nProvider } from '$lib/temporal';
	import TemporalWorkflow from './temporal-workflow.svelte';
	import type { HistoryEvent } from '@forkbombeu/temporal-ui';
	import {
		setupEmitter,
		setupListener,
		type PageMessage,
		type IframeMessage
	} from '../_partials/page-events';
	import type { WorkflowExecution } from '@forkbombeu/temporal-ui/dist/types/workflows';

	//

	let execution = $state<WorkflowExecution>();
	let eventHistory = $state<HistoryEvent[]>();

	setupListener<PageMessage>((ev) => {
		if (ev.type === 'workflow') {
			execution = ev.execution;
			eventHistory = ev.eventHistory;
		}
	});

	const emit = setupEmitter<IframeMessage>(() => parent);
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
	bind:offsetHeight={null, (data) => emit({ type: 'height', height: data })}
>
	<TemporalI18nProvider>
		{#if execution && eventHistory}
			<TemporalWorkflow {execution} {eventHistory} onLoad={() => emit({ type: 'ready' })} />
		{/if}
	</TemporalI18nProvider>
</div>
