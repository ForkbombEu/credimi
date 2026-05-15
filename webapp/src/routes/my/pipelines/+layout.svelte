<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pipeline } from '$lib';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		Pipeline.Runners.status.startPolling(() => Pipeline.Runners.store.read());
		return () => Pipeline.Runners.status.stopPolling();
	});
</script>

{@render children()}
