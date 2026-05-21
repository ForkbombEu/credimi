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
		labelRight?: Snippet;
	};

	let {
		label,
		children,
		class: className,
		required = false,
		optional = false,
		labelRight
	}: Props = $props();
</script>

<div class={['space-y-1', className]}>
	<div class="flex items-center justify-between gap-4">
		<T class="text-sm text-muted-foreground">
			{label}
			{#if required}
				<span class="text-red-500">*</span>
			{/if}
			{#if optional}
				<span class="text-muted-foreground">({m.Optional()})</span>
			{/if}
		</T>
		{@render labelRight?.()}
	</div>

	{@render children?.()}
</div>
