<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import '../app.css';
	import { appName } from '@/brand';
	import { Toaster } from '@/components/ui/sonner';

	let { children } = $props();
</script>

<svelte:head>
	<title>
		{appName}
	</title>
</svelte:head>

{@render children()}

<Toaster richColors closeButton />
