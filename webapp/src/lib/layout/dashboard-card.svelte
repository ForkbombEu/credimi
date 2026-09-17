<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import type { DashboardRecord } from './dashboard-types';
</script>

<script lang="ts" generics="R extends DashboardRecord">
	import type { Snippet } from 'svelte';
	import type { Attachment } from 'svelte/attachments';

	import { ArrowDown, ArrowUp, EllipsisVertical } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import { getCustomCheckPublicUrl } from '$lib/hub/utils';
	import { getPath, mergePaths } from '$lib/utils';
	import { String } from 'effect';
	import { truncate } from 'lodash';
	import removeMd from 'remove-markdown';

	import {
		RecordClone,
		RecordDelete,
		RecordEdit,
		type RecordAction
	} from '@/collections-components/manager';
	import A from '@/components/ui-custom/a.svelte';
	import Avatar from '@/components/ui-custom/avatar.svelte';
	import Card from '@/components/ui-custom/card.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { buttonVariants } from '@/components/ui/button';
	import * as DropdownMenu from '@/components/ui/dropdown-menu';
	import { Separator } from '@/components/ui/separator';
	import { m } from '@/i18n';
	import { Collections, type CustomChecksResponse } from '@/pocketbase/types';

	import LabelLink from './label-link.svelte';
	import PublishedSwitch from './published-switch.svelte';

	//

	type Props = {
		record: R;
		content?: Snippet;
		links?: Record<string, string>;
		avatar?: ((record: R) => string) | string | undefined;
		subtitle?: string;
		badge?: string;
		actions?: Snippet;
		secondaryActions?: Snippet;
		editAction?: Snippet;
		publishAction?: Snippet;
		nameRight?: Snippet;
		afterDescription?: Snippet;
		hideActions?: (RecordAction | 'publish')[] | true;
		hideSeparator?: boolean;
	};

	let {
		record,
		content,
		links = {},
		avatar,
		subtitle,
		badge,
		actions,
		secondaryActions,
		editAction,
		publishAction,
		nameRight,
		hideActions = [],
		afterDescription,
		hideSeparator = false
	}: Props = $props();

	//

	const linkEntries = $derived(Object.entries(links));
	const description = $derived(removeMd(record.description ?? ''));
	const maxDescriptionLength = 400;
	const truncatedDescription = $derived(truncate(description, { length: maxDescriptionLength }));
	const shouldTruncateDescription = $derived(description.length > maxDescriptionLength);
	let isDescriptionExpanded = $state(false);

	//

	let publicUrl = $derived.by(() => {
		if (!record.published) {
			return undefined;
		} else if (record.collectionName === Collections.CustomChecks) {
			return getCustomCheckPublicUrl(record as CustomChecksResponse);
		} else {
			return resolve('/(public)/hub/[...path]', {
				path: mergePaths(record.collectionName, getPath(record))
			});
		}
	});

	const avatarSrc = $derived.by(() => {
		if (typeof avatar === 'function') return avatar(record);
		return avatar;
	});

	//

	const hideActionsList = $derived(hideActions === true ? ['all'] : hideActions);

	const showPublish = $derived(!hideActionsList.includes('publish'));
	const showClone = $derived(!hideActionsList.includes('clone'));
	const showEdit = $derived(Boolean(editAction) || !hideActionsList.includes('edit'));
	const showDelete = $derived(!hideActionsList.includes('delete'));
	const hasSecondaryControls = $derived(
		showPublish || Boolean(secondaryActions) || showClone || showEdit || showDelete
	);

	/** Shared with Run now `@[48rem]` — inline secondary vs overflow menu. */
	const secondaryInlineMinWidthPx = 48 * 16;

	let isCompactHeader = $state(true);

	const observeHeaderCompact: Attachment = (el) => {
		const update = () => {
			isCompactHeader = el.clientWidth < secondaryInlineMinWidthPx;
		};

		update();
		if (typeof ResizeObserver === 'undefined') return;

		const observer = new ResizeObserver(update);
		observer.observe(el);
		return () => observer.disconnect();
	};
</script>

<Card
	id={record.canonified_name}
	class="@container scroll-mt-5 rounded-sm bg-card"
	contentClass="space-y-3 p-4"
>
	<div {@attach observeHeaderCompact} class="flex items-center justify-between gap-3">
		<div class="flex min-w-0 flex-1 items-center gap-4">
			<Avatar src={avatarSrc} fallback={record.name} class="shrink-0 rounded-sm border" />
			<div class="min-w-0 space-y-1">
				<div class="flex min-w-0 items-center gap-2">
					<LabelLink
						label={record.name}
						href={publicUrl}
						published={record.published}
						textToCopy={getPath(record)}
					/>
					{#if nameRight}
						<div class="shrink-0">
							{@render nameRight()}
						</div>
					{/if}
					{#if badge}
						<Badge variant="secondary" class="shrink-0">{badge}</Badge>
					{/if}
				</div>
				{#if subtitle}
					<T class="block truncate text-xs text-gray-400">{subtitle}</T>
				{/if}
			</div>
		</div>

		{#if hideActions !== true}
			<div class="flex shrink-0 items-center gap-2">
				{@render actions?.()}

				{#if hasSecondaryControls}
					{#if isCompactHeader}
						<DropdownMenu.Root>
							<DropdownMenu.Trigger
								class={buttonVariants({ variant: 'outline', size: 'icon' })}
							>
								<EllipsisVertical class="size-4" aria-hidden="true" />
								<span class="sr-only">{m.Actions()}</span>
							</DropdownMenu.Trigger>
							<DropdownMenu.Content
								align="end"
								class="flex max-w-xs flex-wrap items-center gap-2 p-2"
							>
								{@render secondaryControls()}
							</DropdownMenu.Content>
						</DropdownMenu.Root>
					{:else}
						<div class="flex items-center gap-2">
							{@render secondaryControls()}
						</div>
					{/if}
				{/if}
			</div>
		{/if}
	</div>

	{#if String.isNonEmpty(description) || Boolean(links?.length) || afterDescription}
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
							class="inline-flex items-baseline gap-0.5 text-primary hover:underline"
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

			{@render afterDescription?.()}
		</div>
	{/if}

	{#if content}
		{#if !hideSeparator}
			<Separator />
		{/if}
		{@render content()}
	{/if}
</Card>

{#snippet secondaryControls()}
	{#if showPublish}
		{#if publishAction}
			{@render publishAction()}
		{:else}
			<PublishedSwitch record={record as DashboardRecord} field="published" />
		{/if}
	{/if}

	{@render secondaryActions?.()}

	{#if showClone}
		<RecordClone collectionName={record.collectionName} recordId={record.id} size="md" />
	{/if}

	{#if editAction}
		{@render editAction()}
	{:else if showEdit}
		<!-- eslint-disable-next-line @typescript-eslint/no-explicit-any -->
		<RecordEdit record={record as any} />
	{/if}

	{#if showDelete}
		<!-- eslint-disable-next-line @typescript-eslint/no-explicit-any -->
		<RecordDelete record={record as any} />
	{/if}
{/snippet}

{#snippet infoLink(props: { label: string; href?: string | null })}
	<div class="flex items-center gap-1">
		<T class="text-nowrap">{props.label}:</T>
		{#if props.href}
			<A
				class="block w-0 grow cursor-pointer truncate text-gray-400! underline underline-offset-2"
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
