<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Pipeline } from '$lib';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => Pipeline.Runner.Catalog.startLiveRefresh());
</script>

{@render children()}
