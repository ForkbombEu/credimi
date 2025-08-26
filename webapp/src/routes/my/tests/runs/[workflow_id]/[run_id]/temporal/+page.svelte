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
	<style>
		/* Override temporal CSS to ensure scrollbars follow app theme, not temporal theme */
		::-webkit-scrollbar {
			width: 10px !important;
			height: 10px !important;
		}

		::-webkit-scrollbar-track {
			background: transparent !important;
		}

		/* Light theme scrollbar (default) */
		::-webkit-scrollbar-thumb {
			background: hsl(245, 17%, 90%) !important; /* Your app's light mode border color */
			border-radius: 6px !important;
		}

		::-webkit-scrollbar-thumb:hover {
			background: hsl(245, 17%, 80%) !important;
		}

		/* Dark theme scrollbar when .dark class is present */
		.dark ::-webkit-scrollbar-thumb {
			background: hsl(12, 6.5%, 15.1%) !important; /* Your app's dark mode border color */
		}

		.dark ::-webkit-scrollbar-thumb:hover {
			background: hsl(12, 6.5%, 25%) !important;
		}

		/* Reset temporal's border color override to not affect other elements */
		* {
			border-color: revert !important;
		}

		/* Restore your app's border styling */
		body * {
			border-color: hsl(var(--border)) !important;
		}
	</style>
</svelte:head>

<div
	id="temporal-workflow-container"
	class="bg-temporal block"
	bind:offsetHeight={null, (data) => emit({ type: 'height', height: data })}
>
	<TemporalI18nProvider>
		{#if execution && eventHistory}
			<TemporalWorkflow {execution} {eventHistory} onLoad={() => emit({ type: 'ready' })} />
		{/if}
	</TemporalI18nProvider>
</div>

<style lang="postcss">
	.bg-temporal {
		background-color: rgb(248 250 252);
	}

	.padding-x {
		@apply !px-2 md:!px-4 lg:!px-8;
	}
</style>
