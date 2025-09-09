<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import PlainCard from '@/components/ui-custom/plainCard.svelte';
	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		title?: string;
		description?: string;
		children?: Snippet;
		headerActions?: Snippet;
		class?: string;
	};

	let { title, description, children, headerActions, class: className = '' }: Props = $props();
</script>

<PlainCard class="space-y-6 !p-6 md:space-y-8 md:!p-8 {className}">
	{#if title || description}
		<div class="space-y-1 border-b pb-2">
			<div class="flex items-start justify-between gap-4">
				<div class="min-w-0 flex-1">
					{#if title}
						<T tag="h4">
							{title}
						</T>
					{/if}
					{#if description}
						<T class="text-muted-foreground ">
							{description}
						</T>
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
</PlainCard>
