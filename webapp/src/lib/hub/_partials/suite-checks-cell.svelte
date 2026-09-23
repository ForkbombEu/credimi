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
	<span class="text-xs text-muted-foreground">
		{suite.check_count}
		{m.Tests()}
	</span>
{:else if suite.check_files.length > 0}
	<ChildrenCell
		items={suite.check_paths.map((p, i) => ({
			title: suite.check_titles[i] || suite.check_files[i]?.replace('.yaml', '') || p,
			href: `/hub/conformance-checks/${p}`
		}))}
	/>
{:else}
	<span class="text-xs text-muted-foreground">
		{suite.check_count}
		{m.Checks()}
	</span>
{/if}
