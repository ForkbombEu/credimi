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
	import { resolve } from '$app/paths';

	import Avatar from '@/components/ui-custom/avatar.svelte';

	let { value }: Column.Props<typeof column> = $props();

	function lookupSuiteMeta(standardUid: string, versionUid: string, suiteUid: string) {
		const std = Conformance.Standards.Store.get().standards.find((s) => s.uid === standardUid);
		const ver = std?.versions.find((v) => v.uid === versionUid);
		const suite = ver?.suites.find((su) => su.uid === suiteUid);
		return {
			logo: suite?.logo,
			name: suite?.name ?? suiteUid
		};
	}
</script>

<div class="flex flex-col gap-2">
	{#each value as suite (suite.standardUid + suite.versionUid + suite.suiteUid)}
		{@const meta = lookupSuiteMeta(suite.standardUid, suite.versionUid, suite.suiteUid)}
		{@const suiteHref = localizeHref(
			Marketplace.Conformance.getSuitePageUrl(
				suite.standardUid,
				suite.versionUid,
				suite.suiteUid
			)
		)}
		<div>
			<div class="flex items-start gap-2">
				<a
					href={resolve(suiteHref as '/')}
					class="w-fit shrink-0 rounded-sm hover:ring-2 hover:ring-primary"
				>
					<Avatar
						src={meta.logo}
						fallback={meta.name.slice(0, 2)}
						alt={meta.name}
						class="size-8 rounded-sm border bg-muted uppercase"
					/>
				</a>
				<div class="flex flex-col">
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a class="text-xs font-bold text-primary hover:underline" href={suiteHref}>
						{suite.title}
					</a>
					<ul class="flex list-disc flex-col">
						{#each suite.checks as check (check.path)}
							{@const checkHref = localizeHref(
								Marketplace.Conformance.getStandardCheckUrlFromPath(check.path)
							)}
							<li class="max-w-[35ch] truncate text-xs">
								<a
									class="text-primary hover:underline"
									href={resolve(checkHref as '/')}
								>
									{check.id}
								</a>
							</li>
						{/each}
					</ul>
				</div>
			</div>
		</div>
	{:else}
		<Na />
	{/each}
</div>
