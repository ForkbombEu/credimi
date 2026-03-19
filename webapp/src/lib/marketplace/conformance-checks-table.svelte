<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { StandardsWithTestSuites } from '$lib/standards';

	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import ChildrenCell from './_partials/table-children-cell.svelte';
	import TableNameCell from './_partials/table-name-cell.svelte';

	//

	type Props = {
		standardsWithTestSuites: StandardsWithTestSuites;
	};

	let { standardsWithTestSuites }: Props = $props();

	const rows = $derived(
		standardsWithTestSuites.flatMap((standard) =>
			standard.versions.flatMap((version) =>
				version.suites.map((suite) => ({ standard, version, suite }))
			)
		)
	);
</script>

<Table.Table>
	<Table.Header>
		<Table.Head class="px-4">{m.Standard()}</Table.Head>
		<Table.Head class="px-4">{m.Version()}</Table.Head>
		<Table.Head class="px-4">{m.Suite()}</Table.Head>
		<Table.Head class="px-4">{m.Checks()}</Table.Head>
	</Table.Header>
	<Table.Body>
		{#each rows as { standard, version, suite } (standard.uid + version.uid + suite.uid)}
			<Table.Row>
				<Table.Cell class="px-4">
					<TableNameCell
						name={standard.name}
						href={`/marketplace/conformance-checks/${standard.uid}/${version.uid}/${suite.uid}`}
						logo={suite.logo}
					/>
				</Table.Cell>
				<Table.Cell class="px-4">{version.name}</Table.Cell>
				<Table.Cell class="px-4">{suite.name}</Table.Cell>
				<Table.Cell class="px-4">
					{#if suite.files.length > 0}
						<ChildrenCell
							items={suite.paths.map((p, i) => ({
								title: suite.files[i].replace('.yaml', ''),
								href: `/marketplace/conformance-checks/${p}`
							}))}
						/>
					{/if}
				</Table.Cell>
			</Table.Row>
		{/each}
	</Table.Body>
</Table.Table>
