<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import A from '@/components/ui-custom/a.svelte';

	import Na from './na.svelte';
	import { getRelatedEntityHref, type RelatedEntity } from './types';

	//

	type Props = {
		records: RelatedEntity[];
	};

	let { records }: Props = $props();
</script>

<div class={{ 'flex h-[30px] items-center': records.length == 1 }}>
	{#each records as record (record.id)}
		<A href={getRelatedEntityHref(record)} class="block max-w-[30ch] truncate text-xs">
			{record.name}
		</A>
	{:else}
		<Na />
	{/each}
</div>
