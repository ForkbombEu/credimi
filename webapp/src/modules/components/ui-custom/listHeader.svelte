<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import T from './t.svelte';

	interface Props {
		label?: string | undefined;
		class?: string;
		children?: Snippet;
	}

	let { label = undefined, class: className = '', children }: Props = $props();
</script>

<div class="bg-primary-foreground pl-3 {className}">
	{#if children}{@render children()}{:else if label}
		<T tag="small">{label}</T>
	{/if}
</div>
