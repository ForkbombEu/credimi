<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	type DashboardCardRecord = {
		id: string;
		collectionId: string;
		published: boolean;
		name: string;
		description: string | undefined;
		canonified_name: string | undefined;
	};
</script>

<script lang="ts" generics="R extends DashboardCardRecord">
	import type { Snippet } from 'svelte';

	import { String } from 'effect';
	import removeMd from 'remove-markdown';

	import type { OrganizationsRecord } from '@/pocketbase/types';

	import { RecordDelete, RecordEdit } from '@/collections-components/manager';
	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Card from '@/components/ui-custom/card.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { Separator } from '@/components/ui/separator';

	import LabelLink from './label-link.svelte';
	import PublishedSwitch from './published-switch.svelte';

	type Props = {
		record: R;
		content?: Snippet;
		links?: Record<string, string>;
		avatar: (record: R) => string;
		organization: OrganizationsRecord;
		subtitle?: string;
		badge?: string;
		actions?: Snippet;
		editAction?: Snippet;
	};

	let {
		record,
		content,
		links = {},
		avatar,
		organization,
		subtitle,
		badge,
		actions,
		editAction
	}: Props = $props();

	const description = $derived(removeMd(record.description ?? ''));
	const linkEntries = $derived(Object.entries(links));
</script>

<Card id={record.canonified_name} class="bg-card" contentClass="space-y-4 p-4 scroll-mt-5">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-4">
			<Avatar src={avatar(record)} fallback={record.name} class="rounded-sm border" />
			<div>
				<div class="flex items-center gap-2">
					<LabelLink
						label={record.name}
						href="/marketplace/verifiers/{organization?.canonified_name}/{record.canonified_name}"
						published={record.published}
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
			<PublishedSwitch record={record as DashboardCardRecord} field="published" />
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
					{description}
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
