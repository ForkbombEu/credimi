<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	import type { BaseSystemFields } from '@/pocketbase/types';
</script>

<script lang="ts" generics="R extends BaseSystemFields">
	import type { Snippet } from 'svelte';

	import { path as makePath } from '$lib/utils';

	import type { StringKey } from '@/utils/types';

	import { RecordClone, RecordDelete, RecordEdit } from '@/collections-components/manager';
	import IconButton from '@/components/ui-custom/iconButton.svelte';

	import LabelLink from './label-link.svelte';
	import PublishedSwitch from './published-switch.svelte';

	//

	type Props = {
		record: R;
		nameField: StringKey<R>;
		fallbackNameField?: StringKey<R>;
		publicUrl?: string;
		actions?: Snippet;
		hideClone?: boolean;
		path: string[];
	};

	let { record, nameField, fallbackNameField, publicUrl, actions, hideClone, path }: Props =
		$props();

	const name = $derived(
		// @ts-expect-error - Slight type mismatch
		(record[nameField] as string) || (record[fallbackNameField as string] as string)
	);

	const published = $derived(
		'published' in record && typeof record.published === 'boolean' ? record.published : false
	);
</script>

<li class="bg-muted flex items-center justify-between gap-4 rounded-md p-2 pl-3 pr-2 hover:ring-2 hover:ring-blue-200">
	<LabelLink label={name} href={publicUrl} {published} textToCopy={makePath(path)} />

	<div class="flex items-center gap-2">
		{@render actions?.()}

		{#if 'published' in record && typeof record.published === 'boolean'}
			<PublishedSwitch
				record={record as BaseSystemFields & { published: boolean }}
				size="sm"
				field="published"
			/>
		{/if}

		{#if !hideClone}
			<RecordClone collectionName={record.collectionName} recordId={record.id} />
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
