<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { mode } from 'mode-watcher';
	import { Toaster as Sonner, type ToasterProps as SonnerProps } from 'svelte-sonner';

	let restProps: SonnerProps = $props();
</script>

<Sonner theme={$mode} class="toaster group" {...restProps} />

<style global>
	/* Override Sonner toast action button to match default button style */
	:global([data-sonner-toast][data-styled='true'] [data-button]) {
		background: hsl(var(--primary)) !important;
		color: hsl(var(--primary-foreground)) !important;
		border: none !important;
		height: 32px !important;
		padding: 0 12px !important;
		font-size: 14px !important;
		border-radius: 6px !important;
		font-weight: 500 !important;
		transition: all 0.2s !important;
	}

	:global([data-sonner-toast][data-styled='true'] [data-button]:hover) {
		background: hsl(var(--primary) / 0.9) !important;
	}
</style>
