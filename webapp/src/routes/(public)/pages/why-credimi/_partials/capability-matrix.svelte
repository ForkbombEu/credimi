<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { columns, type Matrix } from '$lib/content/why-credimi';

	import SupportDot from './support-dot.svelte';

	const { matrix }: { matrix: Matrix } = $props();
</script>

<section aria-labelledby="matrix-{matrix.id}">
	<h3 id="matrix-{matrix.id}" class="text-2xl font-bold tracking-[-0.02em] text-primary">
		{matrix.title}
	</h3>

	<div class="mt-5 overflow-x-auto rounded-[10px] border border-border bg-card">
		<table class="w-full min-w-[44rem] border-collapse text-left">
			<thead>
				<tr>
					<th
						scope="col"
						class="sticky left-0 z-10 min-w-[12rem] bg-card px-5 py-4 align-bottom"
					>
						<span class="sr-only">Capability</span>
					</th>
					{#each matrix.columns as id (id)}
						{@const column = columns[id]}
						<th
							scope="col"
							class={[
								'min-w-[8rem] px-3 py-4 text-center align-bottom md:w-[11rem]',
								id === 'credimi' && 'bg-primary text-primary-foreground'
							]}
						>
							<span class="block text-sm leading-snug font-bold">
								{#each column.products as product, i (product)}
									{product}{#if i < column.products.length - 1}<br />{/if}
								{/each}
							</span>
							<span
								class={[
									'mt-1 block text-xs font-medium',
									id === 'credimi'
										? 'text-primary-foreground/80'
										: 'text-slate-600'
								]}
							>
								{column.category}
							</span>
						</th>
					{/each}
				</tr>
			</thead>
			<tbody>
				{#each matrix.rows as row (row.label)}
					<tr
						class="border-t border-border transition-colors duration-150 hover:bg-secondary/50"
					>
						<th
							scope="row"
							class="sticky left-0 z-10 bg-card px-5 py-4 text-base leading-snug font-medium text-foreground"
						>
							{row.label}
						</th>
						{#each row.values as value, i (i)}
							{@const isCredimi = matrix.columns[i] === 'credimi'}
							<td class={['px-3 py-4 text-center', isCredimi && 'bg-primary/[0.06]']}>
								<SupportDot {value} />
								{#if isCredimi && row.credimiNote}
									<span class="mt-1 block font-mono text-xs text-primary">
										{row.credimiNote}
									</span>
								{/if}
							</td>
						{/each}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	{#if matrix.footnote}
		<p class="mt-3 max-w-[75ch] text-sm leading-relaxed text-slate-600">{matrix.footnote}</p>
	{/if}
</section>
