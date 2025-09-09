<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import Footer from '$lib/layout/footer.svelte';
	import Topbar from '$lib/layout/topbar.svelte';

	let { children }: { children: Snippet } = $props();
</script>

<div class="flex min-h-screen flex-col">
	<Topbar />
	<div class="flex grow flex-col">
		{@render children()}
	</div>
	<Footer />
</div>
