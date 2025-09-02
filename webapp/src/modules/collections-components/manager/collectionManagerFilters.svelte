<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import type { FilterMode } from '@/pocketbase/query/query';
	import type { GenericRecord } from '@/utils/types';

	import T from '@/components/ui-custom/t.svelte';
	import { Button, buttonVariants, type ButtonVariant } from '@/components/ui/button';
	import { Checkbox } from '@/components/ui/checkbox';
	import { Label } from '@/components/ui/label';
	import * as Popover from '@/components/ui/popover/index.js';
	import * as Sheet from '@/components/ui/sheet/index.js';
	import { m } from '@/i18n';
	import { ensureArray } from '@/utils/other';

	import { getCollectionManagerContext, type Filter } from './collectionManagerContext';

	//

	type Props = {
		modalType?: 'popover' | 'sheet';
		triggerVariant?: ButtonVariant;
		children?: Snippet;
		beforeFilters?: Snippet;
		afterFilters?: Snippet;
		trigger?: Snippet<[{ props: GenericRecord }]>;
	};

	let {
		children,
		modalType = 'popover',
		triggerVariant = 'outline',
		beforeFilters,
		afterFilters,
		trigger
	}: Props = $props();

	const { filters, manager } = getCollectionManagerContext();
</script>

{#if modalType === 'popover'}
	<Popover.Root>
		<Popover.Trigger class={buttonVariants({ variant: triggerVariant })}>
			{#snippet child({ props })}
				{@render childSnippet(props)}
			{/snippet}
		</Popover.Trigger>

		<Popover.Content class="w-80">
			{@render content()}
		</Popover.Content>
	</Popover.Root>
{/if}

{#if modalType === 'sheet'}
	<Sheet.Root>
		<Sheet.Trigger class={buttonVariants({ variant: triggerVariant })}>
			{#snippet child({ props })}
				{@render childSnippet(props)}
			{/snippet}
		</Sheet.Trigger>

		<Sheet.Content>
			{@render content()}
		</Sheet.Content>
	</Sheet.Root>
{/if}

{#snippet content()}
	{@render beforeFilters?.()}
	<ul class="space-y-4">
		{#each ensureArray(filters) as filterGroup}
			<ul class="space-y-2">
				<li><T class="font-bold">{filterGroup.name ?? m.filters()}</T></li>
				{#each filterGroup.filters as filter}
					<li>
						{@render FilterInput(filter, filterGroup.id, filterGroup.mode)}
					</li>
				{/each}
			</ul>
		{/each}
	</ul>

	{@render afterFilters?.()}
{/snippet}

{#snippet FilterInput(f: Filter, id: string, mode: FilterMode)}
	<div class="flex items-center gap-2">
		<Checkbox
			id={f.name}
			name={f.name}
			value={f.expression}
			checked={manager.query.hasFilter(f.expression, id)}
			onCheckedChange={(v) => {
				if (v) {
					manager.query.addFilter(f.expression, id, mode);
				} else {
					manager.query.removeFilter(f.expression, id);
				}
			}}
		/>
		<Label for={f.name} class="text-md hover:cursor-pointer">{f.name}</Label>
	</div>
{/snippet}

{#snippet childSnippet(props: GenericRecord)}
	{#if trigger}
		{@render trigger({ props })}
	{:else}
		<Button {...props} variant={triggerVariant}>
			{#if children}
				{@render children()}
			{:else}
				{m.Filters()}
			{/if}
		</Button>
	{/if}
{/snippet}
