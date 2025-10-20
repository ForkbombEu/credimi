<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script module lang="ts">
	type BaseItem = {
		id: string;
		collectionName: string;
		collectionId: string;
		canonified_name: string;
		published: boolean;
		name: string;
	};
</script>

<script lang="ts" generics="R extends BaseItem">
	import type { OrganizationsRecord } from '@/pocketbase/types';

	import { RecordClone, RecordDelete, RecordEdit } from '@/collections-components/manager';
	import CopyButtonSmall from '@/components/ui-custom/copy-button-small.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import { m } from '@/i18n';

	import LabelLink from './label-link.svelte';
	import PublishedSwitch from './published-switch.svelte';

	//

	type Props = {
		record: R;
		organization: OrganizationsRecord;
		pathItems?: string[];
	};

	let { records, organization, pathItems = [] }: Props = $props();

	const path = $derived(pathItems.filter(Boolean).join('/'));
</script>

<li class="bg-muted flex items-center justify-between rounded-md p-2 pl-3 pr-2">
	<LabelLink
		label={record.name}
		href="/marketplace/{record.collectionName}/{organization.canonified_name}/{record.canonified_name}"
		published={record.published}
	/>

	<div class="flex items-center gap-2">
		<Tooltip>
			<CopyButtonSmall textToCopy={path} square />
			{#snippet content()}
				<p>{m.Copy()}: {path}</p>
			{/snippet}
		</Tooltip>

		<PublishedSwitch record={record as BaseItem} size="sm" field="published" />

		<RecordClone collectionName="use_cases_verifications" record={record as never} />

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
