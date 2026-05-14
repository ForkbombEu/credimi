<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pipeline } from '$lib';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		Pipeline.Runner.status.startPolling(() => Pipeline.Runner.store.read());
		return () => Pipeline.Runner.status.stopPolling();
	});
</script>

{@render children()}
