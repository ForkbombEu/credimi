<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import type { DashboardRecord } from './dashboard-types';
</script>

<script lang="ts" generics="R extends DashboardRecord">
	import type { Snippet } from 'svelte';

	import { getMarketplaceItemUrl, type MarketplaceItem } from '$lib/marketplace';
	import { path as makePath } from '$lib/utils';
	import { String } from 'effect';
	import { truncate } from 'lodash';
	import { ArrowDown, ArrowUp } from 'lucide-svelte';
	import removeMd from 'remove-markdown';

	import { RecordDelete, RecordEdit } from '@/collections-components/manager';
	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Card from '@/components/ui-custom/card.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { Separator } from '@/components/ui/separator';
	import { pb } from '@/pocketbase';

	import LabelLink from './label-link.svelte';
	import PublishedSwitch from './published-switch.svelte';

	//

	type Props = {
		record: R;
		content?: Snippet;
		links?: Record<string, string>;
		avatar: (record: R) => string;
		subtitle?: string;
		badge?: string;
		actions?: Snippet;
		editAction?: Snippet;
		path: string[];
	};

	let {
		record,
		content,
		links = {},
		avatar,
		subtitle,
		badge,
		actions,
		editAction,
		path
	}: Props = $props();

	//

	const linkEntries = $derived(Object.entries(links));
	const description = $derived(removeMd(record.description ?? ''));
	const maxDescriptionLength = 400;
	const truncatedDescription = $derived(truncate(description, { length: maxDescriptionLength }));
	const shouldTruncateDescription = $derived(description.length > maxDescriptionLength);
	let isDescriptionExpanded = $state(false);

	//

	let publicUrl = $state('');
	$effect(() => {
		pb.collection('marketplace_items')
			.getOne(record.id)
			.then((item) => {
				publicUrl = getMarketplaceItemUrl(item as unknown as MarketplaceItem);
			})
			.catch((error) => {
				console.warn(
					"Probably not a marketplace item since it's not published",
					'\n-\n',
					error
				);
			});
	});
</script>

<Card id={record.canonified_name} class="bg-card scroll-mt-5" contentClass="space-y-4 p-4">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-4">
			<Avatar src={avatar(record)} fallback={record.name} class="rounded-sm border" />
			<div>
				<div class="flex items-center gap-2">
					<LabelLink
						label={record.name}
						href={publicUrl}
						published={record.published}
						textToCopy={makePath(path)}
					/>
					{#if badge}
						<Badge variant="secondary">{badge}</Badge>
					{/if}
				</div>
				{#if subtitle}
					<T class="block truncate text-xs text-gray-400">{subtitle}</T>
				{/if}
			</div>
		</div>
		<div class="flex items-center gap-2">
			{@render actions?.()}
			<PublishedSwitch record={record as DashboardRecord} field="published" />
			{#if editAction}
				{@render editAction()}
			{:else}
				<!-- eslint-disable-next-line @typescript-eslint/no-explicit-any -->
				<RecordEdit record={record as any} />
			{/if}
			<!-- eslint-disable-next-line @typescript-eslint/no-explicit-any -->
			<RecordDelete record={record as any} />
		</div>
	</div>

	{#if String.isNonEmpty(description) || Boolean(links?.length)}
		<Separator />

		<div class="space-y-3 text-xs">
			{#if String.isNonEmpty(description)}
				<T class="mt-0.5 leading-normal text-gray-400">
					{#if !shouldTruncateDescription}
						{description}
					{:else if shouldTruncateDescription}
						{#if isDescriptionExpanded}
							{description}
						{:else}
							{truncatedDescription}
						{/if}
					{/if}
					{#if shouldTruncateDescription}
						{@const icon = isDescriptionExpanded ? ArrowUp : ArrowDown}
						{@const label = isDescriptionExpanded ? 'Collapse' : 'Expand'}
						<button
							class="text-primary inline-flex items-baseline gap-0.5 hover:underline"
							onclick={() => (isDescriptionExpanded = !isDescriptionExpanded)}
						>
							<Icon src={icon} size="14" class="translate-y-0.5" />
							{label}
						</button>
					{/if}
				</T>
			{/if}

			{#if linkEntries.length}
				<div class="space-y-1">
					{#each Object.entries(links) as link (link)}
						{@render infoLink({ label: link[0], href: link[1] })}
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	{#if content}
		<Separator />
		{@render content()}
	{/if}
</Card>

{#snippet infoLink(props: { label: string; href?: string | null })}
	<div class="flex items-center gap-1">
		<T class="text-nowrap">{props.label}:</T>
		{#if props.href}
			<A
				class="block w-0 grow cursor-pointer truncate !text-gray-400 underline underline-offset-2"
				target="_blank"
				href={props.href}
			>
				{props.href}
			</A>
		{:else}
			<T class="text-gray-400">-</T>
		{/if}
	</div>
{/snippet}
