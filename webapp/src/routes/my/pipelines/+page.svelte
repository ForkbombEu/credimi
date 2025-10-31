<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import DashboardCard from '$lib/layout/dashboard-card.svelte';

	import { CollectionManager } from '@/collections-components';
	import { pb } from '@/pocketbase';

	let { data } = $props();
	let { organization } = $derived(data);
</script>

<CollectionManager collection="pipelines">
	{#snippet records({ records })}
		<div class="space-y-6">
			{#each records as pipeline (pipeline.id)}
				<DashboardCard
					record={pipeline}
					avatar={() => pb.files.getURL(organization, organization.logo)}
				/>
			{/each}
		</div>
	{/snippet}
</CollectionManager>
