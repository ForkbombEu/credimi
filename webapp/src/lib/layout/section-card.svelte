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
		removeFlexRestrictions?: boolean;
		headerHasFlexWrap?: boolean;
	};

	const {
		children,
		class: className,
		title,
		headerActions,
		subtitle,
		removeFlexRestrictions = false,
		headerHasFlexWrap = false,
		...restProps
	}: Props = $props();
</script>

<div
	class={['scroll-mt-4 space-y-6 rounded-md bg-background p-6 shadow-sm', className]}
	{...restProps}
>
	{#if title}
		<div class="border-b pb-3">
			<div
				class={['flex items-start justify-between gap-4', headerHasFlexWrap && 'flex-wrap']}
			>
				<div class={[!removeFlexRestrictions && 'min-w-0 flex-1']}>
					<T tag="h4" class="overflow-auto">{title}</T>
					{#if subtitle}
						<T class="mt-1 text-sm text-muted-foreground">{subtitle}</T>
					{/if}
				</div>
				{#if headerActions}
					<div class={[!removeFlexRestrictions && 'shrink-0']}>
						{@render headerActions()}
					</div>
				{/if}
			</div>
		</div>
	{/if}
	{@render children?.()}
</div>
