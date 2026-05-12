<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { Conformance } from '$lib';
	import { entities } from '$lib/global';
	import { Marketplace } from '$lib/marketplace';

	import { renderComponent } from '@/components/ui/data-table';
	import { localizeHref } from '@/i18n';

	import * as Column from '../column';
	import EntityHeader from './headers/entity-header.svelte';
	import Na from './partials/na.svelte';

	//

	export const column = Column.define({
		id: 'conformance_checks',
		header: renderComponent(EntityHeader, {
			data: entities.conformance_checks,
			plurality: 'plural'
		}),
		fn: (row) => Conformance.Check.groupPathsBySuite(row.conformance_checks ?? []),
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<div>
	{#each value as suite (suite)}
		<p class="text-xs font-bold">{suite.title}</p>
		<ul class="list-inside list-disc">
			{#each suite.checks as check (check.path)}
				{@const href = Marketplace.Conformance.getStandardCheckUrlFromPath(check.path)}
				<li class="max-w-[35ch] truncate text-xs">
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a class="text-primary hover:underline" href={localizeHref(href)}>
						{check.id}
					</a>
				</li>
			{/each}
		</ul>
	{:else}
		<Na />
	{/each}
</div>
