<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	import T from '@/components/ui-custom/t.svelte';

	type Props = HTMLAttributes<HTMLDivElement> & {
		children: Snippet;
		title?: string;
		headerActions?: Snippet;
		subtitle?: string;
	};

	const {
		children,
		class: className,
		title,
		headerActions,
		subtitle,
		...restProps
	}: Props = $props();
</script>

<div
	class={['bg-background scroll-mt-4 space-y-6 rounded-md p-6 shadow-sm', className]}
	{...restProps}
>
	{#if title}
		<div class="border-b pb-3">
			<div class="flex items-start justify-between gap-4">
				<div class="min-w-0 flex-1">
					<T tag="h4" class="overflow-auto">{title}</T>
					{#if subtitle}
						<T class="text-muted-foreground mt-1 text-sm">{subtitle}</T>
					{/if}
				</div>
				{#if headerActions}
					<div class="flex-shrink-0">
						{@render headerActions()}
					</div>
				{/if}
			</div>
		</div>
	{/if}
	{@render children?.()}
</div>
