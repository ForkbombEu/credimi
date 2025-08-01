<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import * as Sheet from '@/components/ui/sheet';
	import type { ComponentProps, Snippet } from 'svelte';
	import { cn } from '@/components/ui/utils';
	import Button from '@/components/ui-custom/button.svelte';
	import { Separator } from '@/components/ui/separator';
	import type { GenericRecord } from '@/utils/types';

	//

	type SheetSide = ComponentProps<typeof Sheet.Content>['side'];

	interface Props {
		side?: SheetSide;
		title?: string | undefined;
		open?: boolean;
		class?: string;
		contentClass?: string;
		trigger?: Snippet<[{ sheetTriggerAttributes: GenericRecord; openSheet: () => void }]>;
		children?: Snippet;
		content?: Snippet<[{ closeSheet: () => void }]>;
		hideTrigger?: boolean;
	}

	let {
		side = 'right',
		title = undefined,
		open: isOpen = $bindable(false),
		class: className = '',
		contentClass = '',
		trigger,
		children,
		content,
		hideTrigger = false
	}: Props = $props();

	//

	function closeSheet() {
		isOpen = false;
	}

	function openSheet() {
		isOpen = true;
	}
</script>

<Sheet.Root bind:open={isOpen}>
	{#if !hideTrigger}
		<Sheet.Trigger>
			{#snippet child({ props })}
				{#if trigger}
					{@render trigger({ sheetTriggerAttributes: props, openSheet })}
				{:else}
					<Button {...props} class="shrink-0" variant="outline">
						{@render children?.()}
					</Button>
				{/if}
			{/snippet}
		</Sheet.Trigger>
	{/if}

	<Sheet.Content
		side="right"
		class="flex {cn({
			'!min-w-[300px] !max-w-none': side == 'right'
		})} flex-col px-0 {className}"
	>
		{#if title}
			<Sheet.Header class="px-6">
				<Sheet.Title>{title}</Sheet.Title>
				<Separator></Separator>
			</Sheet.Header>
		{/if}

		<div class="overflow-y-auto overflow-x-visible px-6 {contentClass}">
			{@render content?.({ closeSheet })}
		</div>
	</Sheet.Content>
</Sheet.Root>
