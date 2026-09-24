<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ConformanceSuiteRecord } from '$lib/conformance';
	import { annotateNestedHubItemsForSearch } from '$lib/hub/nested-hub-search';

	import { m } from '@/i18n';

	import ChildrenCell from './table-children-cell.svelte';

	//

	type Props = {
		suite: ConformanceSuiteRecord;
		search?: string;
	};

	let { suite, search = '' }: Props = $props();

	const memberLinks = $derived.by(() => {
		if (suite.members.length === 0) return [];

		const items = suite.members.map((member) => {
			const title = member.title || member.file.replace('.yaml', '') || member.path;
			return {
				name: [member.title, member.file, member.path].filter(Boolean).join(' '),
				title,
				href: `/hub/conformance-checks/${member.path}`
			};
		});

		return annotateNestedHubItemsForSearch(items, search).map(({ title, href, dimmed }) => ({
			title,
			href,
			dimmed
		}));
	});
</script>

{#if suite.fs_standard === 'fcaf'}
	<ChildrenCell
		items={[
			{
				title: `${suite.check_count} ${m.Tests()}`,
				href: `/hub/conformance-checks/${suite.path_prefix}`
			}
		]}
	/>
{:else if suite.members.length > 0}
	<ChildrenCell items={memberLinks} />
{:else}
	<span class="text-xs text-muted-foreground">
		{suite.check_count}
		{m.Checks()}
	</span>
{/if}
