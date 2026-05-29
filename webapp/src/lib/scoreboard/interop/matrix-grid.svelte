<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { get as getConformanceStore } from '$lib/conformance';

	import { m } from '@/i18n';

	import type { InteropMatrixResponse } from './types';
	import type { InteropMatrixEntity } from './types';

	import MatrixCell from './matrix-cell.svelte';
	import { resolveConformanceCheck } from './resolve-conformance';

	type Props = {
		matrix: InteropMatrixResponse;
		legend?: Snippet;
	};

	let { matrix, legend }: Props = $props();

	const cellByKey = $derived(
		new Map(matrix.cells.map((cell) => [`${cell.row_id}:${cell.column_id}`, cell] as const))
	);

	const isConformanceColumns = $derived(matrix.column_axis === 'conformance_check');

	const columnCollection = $derived(columnHubCollection(matrix.column_axis));
	const rowCollection = $derived(rowHubCollection(matrix.row_axis));

	function rowHubCollection(axis: string): string {
		return axis === 'use_case_verification' ? 'use_cases_verifications' : 'wallets';
	}

	function columnHubCollection(axis: string): string {
		switch (axis) {
			case 'credential':
				return 'credentials';
			case 'verifier':
				return 'verifiers';
			case 'use_case_verification':
				return 'use_cases_verifications';
			default:
				return 'credential_issuers';
		}
	}

	function enrichedColumn(column: InteropMatrixEntity): InteropMatrixEntity {
		if (!isConformanceColumns) return column;
		const resolved = resolveConformanceCheck(column.id, getConformanceStore().standards);
		if (!resolved) return column;
		return {
			...column,
			name: resolved.name,
			subtitle: resolved.subtitle ?? undefined,
			avatar_url: resolved.avatar_url ?? undefined
		};
	}

	function hubHref(collection: string, path: string) {
		return `/hub/${collection}/${path}`;
	}

	function axisLabel(axis: string): string {
		switch (axis) {
			case 'wallet':
				return m.Wallet();
			case 'issuer':
				return m.Issuer();
			case 'credential':
				return m.Credential();
			case 'verifier':
				return m.Verifier();
			case 'use_case_verification':
				return m.Use_case_verification();
			case 'conformance_check':
				return m.Conformance_check();
			default:
				return axis;
		}
	}

	const cornerLabel = $derived(
		m.interop_matrix_corner_label({
			row: axisLabel(matrix.row_axis),
			column: axisLabel(matrix.column_axis)
		})
	);

	function getSubtitleOrVersion(
		subtitle: string | null | undefined,
		versionLabel: string | null | undefined
	) {
		return subtitle ? subtitle : versionLabel;
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
						{cornerLabel}
					</th>
					{#each matrix.columns as column (column.id)}
						{@const enriched = enrichedColumn(column)}
						{@const columnSubtitle = getSubtitleOrVersion(
							enriched.subtitle,
							enriched.version_label
						)}
						<th
							class="sticky top-0 z-10 min-w-32 border-b bg-muted/60 px-3 py-3 text-center font-semibold"
						>
							<a
								class="inline-flex max-w-44 flex-col items-center gap-1 hover:underline"
								href={isConformanceColumns
									? `/hub/${column.path}`
									: hubHref(columnCollection, column.path)}
							>
								{#if enriched.avatar_url}
									<img
										src={enriched.avatar_url}
										alt={enriched.name}
										class="size-6 rounded-full object-cover"
										loading="lazy"
									/>
								{/if}
								<span>{enriched.name}</span>
								{#if columnSubtitle}
									<span class="text-xs font-normal text-muted-foreground">
										{columnSubtitle}
									</span>
								{/if}
							</a>
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each matrix.rows as row (row.id)}
					{@const rowSubtitle = getSubtitleOrVersion(row.subtitle, row.version_label)}
					<tr class="border-b last:border-b-0">
						<th
							class="sticky left-0 z-10 border-r bg-muted/40 px-3 py-3 text-left align-middle font-medium"
						>
							<a
								class="inline-flex items-center gap-2 hover:underline"
								href={hubHref(rowCollection, row.path)}
							>
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
							{#if rowSubtitle}
								<span
									class="mt-0.5 block text-xs font-normal text-muted-foreground"
								>
									{rowSubtitle}
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
