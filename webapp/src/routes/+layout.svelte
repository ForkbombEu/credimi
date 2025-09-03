<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import GlobalLoading from '$lib/layout/global-loading.svelte';

	import { appName } from '@/brand';
	import { Toaster } from '@/components/ui/sonner';

	import '../app.css';

	//

	let { children } = $props();
</script>

<svelte:head>
	<title>
		{appName}
	</title>
</svelte:head>

{@render children()}

<Toaster richColors closeButton />

<GlobalLoading />
