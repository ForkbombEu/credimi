<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { resolve } from '$app/paths';

	import Avatar from '@/components/ui-custom/avatar.svelte';
	import { localizeHref } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { getRelatedEntityHref, type RelatedEntity } from './types';

	//

	type Props = {
		record: RelatedEntity;
		link?: boolean;
	};

	let { record, link = false }: Props = $props();
</script>

{#if link}
	<a
		href={resolve(localizeHref(getRelatedEntityHref(record)) as '/')}
		class="w-fit hover:ring-2 hover:ring-primary"
	>
		{@render content()}
	</a>
{:else}
	{@render content()}
{/if}

{#snippet content()}
	{#if 'logo' in record}
		<Avatar
			src={pb.files.getURL(record, record.logo)}
			class="size-6 rounded-xs border"
			fallback={record.name.slice(0, 2)}
		/>
	{/if}
{/snippet}
