<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { StandardsWithTestSuites } from '$lib/standards';

	import { CheckCheck } from '@lucide/svelte';

	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import TableNameCell from './_partials/table-name-cell.svelte';
	import TableRowAfter from './_partials/table-row-after.svelte';

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
			</Table.Row>
			{#if suite.files.length > 0}
				<TableRowAfter
					title={m.Checks()}
					show={true}
					items={suite.paths.map((p, i) => ({
						title: suite.files[i].replace('.yaml', ''),
						href: `/marketplace/conformance-checks/${p}`
					}))}
					icon={CheckCheck}
				/>
			{/if}
		{/each}
	</Table.Body>
</Table.Table>
