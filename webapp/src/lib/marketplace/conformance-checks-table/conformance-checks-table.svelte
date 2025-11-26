<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { StandardsWithTestSuites } from '$lib/standards';

	import T from '@/components/ui-custom/t.svelte';
	import * as Table from '@/components/ui/table';
	import { m } from '@/i18n';

	import TableRowAfter from '../table-row-after.svelte';

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
		<Table.Head>{m.Standard()}</Table.Head>
		<Table.Head>{m.Version()}</Table.Head>
		<Table.Head>{m.Suite()}</Table.Head>
	</Table.Header>
	<Table.Body>
		{#each rows as { standard, version, suite } (standard.uid + version.uid + suite.uid)}
			<Table.Row>
				<Table.Cell>
					<div class="flex items-center gap-1">
						<a
							href={`/marketplace/conformance-checks/${standard.uid}/${version.uid}/${suite.uid}`}
							class="hover:underline"
						>
							<T class="overflow-hidden text-ellipsis font-semibold">
								{standard.name}
							</T>
						</a>
						<!-- <CopyButtonSmall textToCopy={typed.path} square variant="ghost" size="xs" /> -->
					</div>
				</Table.Cell>
				<Table.Cell>{version.name}</Table.Cell>
				<Table.Cell>{suite.name}</Table.Cell>
			</Table.Row>
			{#if suite.files.length > 0}
				<TableRowAfter
					title={m.Checks()}
					show={true}
					items={suite.paths.map((p, i) => ({
						title: suite.files[i].replace('.yaml', ''),
						href: `/marketplace/conformance-checks/${p}`
					}))}
				/>
			{/if}
		{/each}
	</Table.Body>
</Table.Table>
