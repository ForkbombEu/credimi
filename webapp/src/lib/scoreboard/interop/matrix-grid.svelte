<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { m } from '@/i18n';

	import MatrixCell from './matrix-cell.svelte';
	import type { InteropMatrixResponse } from './types';

	type Props = {
		matrix: InteropMatrixResponse;
		legend?: Snippet;
	};

	let { matrix, legend }: Props = $props();

	const cellByKey = $derived(
		new Map(matrix.cells.map((cell) => [`${cell.row_id}:${cell.column_id}`, cell] as const))
	);

	function hubHref(collection: 'wallets' | 'credential_issuers', path: string) {
		return `/hub/${collection}/${path}`;
	}
</script>

<div class="mx-auto max-w-7xl px-4 md:px-8">
	{#if legend}
		<div class="mb-4 flex flex-wrap items-center justify-end gap-4">{@render legend()}</div>
	{/if}

	<div class="overflow-x-auto rounded-lg border bg-background shadow-sm">
		<table class="w-full min-w-[640px] border-collapse text-sm">
			<thead>
				<tr>
					<th
						class="sticky left-0 z-20 min-w-40 border-r border-b bg-muted/80 px-3 py-3 text-left text-xs font-semibold tracking-wide text-muted-foreground uppercase"
					>
						{m.interop_matrix_corner_label()}
					</th>
					{#each matrix.columns as column (column.id)}
						<th
							class="sticky top-0 z-10 min-w-32 border-b bg-muted/60 px-3 py-3 text-center font-semibold"
						>
							<a
								class="inline-flex max-w-44 flex-col items-center gap-1 hover:underline"
								href={hubHref('credential_issuers', column.path)}
							>
								{#if column.avatar_url}
									<img
										src={column.avatar_url}
										alt={column.name}
										class="size-6 rounded-full object-cover"
										loading="lazy"
									/>
								{/if}
								<span>{column.name}</span>
								{#if column.subtitle || column.version_label}
									<span class="text-xs font-normal text-muted-foreground">
										{column.subtitle ?? column.version_label}
									</span>
								{/if}
							</a>
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each matrix.rows as row (row.id)}
					<tr class="border-b last:border-b-0">
						<th
							class="sticky left-0 z-10 border-r bg-muted/40 px-3 py-3 text-left align-middle font-medium"
						>
							<a class="inline-flex items-center gap-2 hover:underline" href={hubHref('wallets', row.path)}>
								{#if row.avatar_url}
									<img
										src={row.avatar_url}
										alt={row.name}
										class="size-6 rounded-full object-cover"
										loading="lazy"
									/>
								{/if}
								<span>{row.name}</span>
							</a>
							{#if row.subtitle || row.version_label}
								<span class="mt-0.5 block text-xs font-normal text-muted-foreground">
									{row.subtitle ?? row.version_label}
								</span>
							{/if}
						</th>
						{#each matrix.columns as column (column.id)}
							<td class="border-l p-2 align-top">
								<MatrixCell cell={cellByKey.get(`${row.id}:${column.id}`)} />
							</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
