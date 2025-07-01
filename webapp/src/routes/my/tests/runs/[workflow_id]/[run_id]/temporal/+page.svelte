<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<!-- 
	! IMPORTANT !
	This page has `@` inside its name in order to "reset" the layout stack.
	Styles imported from the `/temporal.css` file conflict with the global styles.
-->

<script>
	import TemporalI18nProvider from './components/temporal-i18n-provider.svelte';
	import TemporalWorkflow from './components/temporal-workflow.svelte';

	//

	let { data } = $props();
	const { workflowResponse, eventHistory } = data;
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

<TemporalI18nProvider>
	<TemporalWorkflow {workflowResponse} {eventHistory} />
</TemporalI18nProvider>
