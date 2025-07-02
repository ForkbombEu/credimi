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

<script>
	import { TemporalI18nProvider } from '$lib/temporal';
	import TemporalWorkflow from './temporal-workflow.svelte';

	//

	let { data } = $props();
	const { workflow, eventHistory } = data;
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

<div>
	<TemporalI18nProvider>
		<TemporalWorkflow
			workflowResponse={workflow}
			{eventHistory}
			onMount={() => {
				sendHeight(document.body.scrollHeight);
			}}
		/>
	</TemporalI18nProvider>
</div>
