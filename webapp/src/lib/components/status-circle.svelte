<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	type Props = {
		size?: number;
	};

	let { size = 16 }: Props = $props();

	// Calculate proportional sizes based on base size of 16px
	// Base proportions:
	// - Container: 16px (1x)
	// - Halos: 16px (1x)
	// - Pulsing core: 12px (0.75x)
	// - Solid core: 8px (0.5x)
	const haloSize = $derived(size);
	const pulsingCoreSize = $derived(size * 0.75);
	const solidCoreSize = $derived(size * 0.5);
</script>

<span
	class="relative inline-flex items-center justify-center"
	style="width: {size}px; height: {size}px;"
>
	<!-- Extending halo rings with gradient -->
	<span
		class="absolute animate-ping rounded-full bg-green-500 opacity-75"
		style="width: {haloSize}px; height: {haloSize}px; animation-duration: 2s;"
	></span>
	<span
		class="absolute animate-ping rounded-full bg-green-400 opacity-50"
		style="width: {haloSize}px; height: {haloSize}px; animation-duration: 2s; animation-delay: 0.5s;"
	></span>
	<!-- Core pulsing circle -->
	<span
		class="absolute animate-pulse rounded-full bg-green-400"
		style="width: {pulsingCoreSize}px; height: {pulsingCoreSize}px;"
	></span>
	<!-- Solid core -->
	<span
		class="relative z-10 rounded-full bg-green-600 ring-2 ring-green-600/20"
		style="width: {solidCoreSize}px; height: {solidCoreSize}px;"
	></span>
</span>
