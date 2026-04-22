<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import A from '@/components/ui-custom/a.svelte';

	import { getEntityHref, type Entity } from './types';

	//

	type Props = {
		records: Entity[];
	};

	let { records }: Props = $props();
</script>

<div>
	{#each records as record (record.id)}
		<A href={getEntityHref(record)} class="block max-w-[15ch] truncate text-xs">
			{record.name}
		</A>
	{/each}
</div>
