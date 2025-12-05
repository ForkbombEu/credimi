<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import Label from '@/components/ui/label/label.svelte';

	//

	interface Props {
		children: Snippet;
		label: string;
		class?: ClassValue;
		required?: boolean;
	}

	let { children, label, required = false, class: className }: Props = $props();
</script>

<div class="space-y-2">
	<Label>
		{label}
		{#if required}
			<span class="text-destructive">*</span>
		{/if}
	</Label>
	<div class={['rounded-md border border-blue-200 bg-blue-50 p-4', className]}>
		{@render children()}
	</div>
</div>
