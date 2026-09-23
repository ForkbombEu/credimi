<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ConformanceSuiteRecord } from '$lib/conformance';

	import { m } from '@/i18n';

	import ChildrenCell from './table-children-cell.svelte';

	//

	type Props = {
		suite: ConformanceSuiteRecord;
	};

	let { suite }: Props = $props();
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
	<ChildrenCell
		items={suite.members.map((m) => ({
			title: m.title || m.file.replace('.yaml', '') || m.path,
			href: `/hub/conformance-checks/${m.path}`
		}))}
	/>
{:else}
	<span class="text-xs text-muted-foreground">
		{suite.check_count}
		{m.Checks()}
	</span>
{/if}
