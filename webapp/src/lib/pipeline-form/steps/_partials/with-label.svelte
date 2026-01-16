<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	type Props = {
		label: string;
		children?: Snippet;
		class?: string;
		required?: boolean;
		optional?: boolean;
	};

	let { label, children, class: className, required = false, optional = false }: Props = $props();
</script>

<div class={['space-y-1', className]}>
	<T class="text-muted-foreground text-sm">
		{label}
		{#if required}
			<span class="text-red-500">*</span>
		{/if}
		{#if optional}
			<span class="text-muted-foreground">({m.Optional()})</span>
		{/if}
	</T>
	{@render children?.()}
</div>
