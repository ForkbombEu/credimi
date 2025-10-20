<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import type { BaseSystemFields } from '@/pocketbase/types';
</script>

<script lang="ts" generics="R extends BaseSystemFields">
	import type { Snippet } from 'svelte';

	import type { StringKey } from '@/utils/types';

	import { RecordClone, RecordDelete, RecordEdit } from '@/collections-components/manager';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';

	import LabelLink from './label-link.svelte';
	import PublishedSwitch from './published-switch.svelte';

	//

	type Props = {
		record: R;
		nameField: StringKey<R>;
		publicUrl?: string;
		textToCopy?: string;
		actions?: Snippet;
		hideClone?: boolean;
	};

	let { record, nameField, publicUrl, textToCopy, actions, hideClone }: Props = $props();
</script>

<li class="bg-muted flex items-center justify-between rounded-md p-2 pl-3 pr-2 hover:ring-2">
	{#if publicUrl && 'published' in record && typeof record.published === 'boolean'}
		<LabelLink
			label={record[nameField] as string}
			href={publicUrl}
			published={record.published}
		/>
	{:else}
		<T class="font-medium">{record[nameField] as string}</T>
	{/if}

	<div class="flex items-center gap-2">
		{@render actions?.()}

		{#if textToCopy}
			<Tooltip>
				<CopyButtonSmall {textToCopy} square />
				{#snippet content()}
					<p>{m.Copy()}: {textToCopy}</p>
				{/snippet}
			</Tooltip>
		{/if}

		{#if 'published' in record && typeof record.published === 'boolean'}
			<PublishedSwitch
				record={record as BaseSystemFields & { published: boolean }}
				size="sm"
				field="published"
			/>
		{/if}

		{#if !hideClone}
			<RecordClone collectionName={record.collectionName as never} record={record as never} />
		{/if}

		<RecordEdit record={record as never}>
			{#snippet button({ triggerAttributes, icon })}
				<IconButton size="sm" variant="outline" {icon} {...triggerAttributes} />
			{/snippet}
		</RecordEdit>

		<RecordDelete record={record as never}>
			{#snippet button({ triggerAttributes, icon })}
				<IconButton size="sm" variant="outline" {icon} {...triggerAttributes} />
			{/snippet}
		</RecordDelete>
	</div>
</li>
