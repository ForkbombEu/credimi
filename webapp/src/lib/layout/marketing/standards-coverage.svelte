<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Check from '@lucide/svelte/icons/check';

	import { fcafCatalog, standards } from './standards';
</script>

<div>
	<div class="fold-motif rounded-2xl bg-primary p-7 text-primary-foreground lg:p-9">
		<p class="relative text-lg font-semibold text-primary-foreground/85">
			EUDI FCAF: wallet solution, relying party role
		</p>
		<p
			class="relative mt-3 text-[clamp(2rem,3.4vw,3rem)] leading-none font-bold tracking-[-0.03em]"
		>
			{fcafCatalog.testCases} test cases
		</p>
		<p class="relative mt-2 text-xl text-primary-foreground/85">
			in {fcafCatalog.scenarios} scenarios
		</p>
	</div>
	<ul class="mt-4 divide-y divide-border rounded-2xl border border-border bg-card">
		{#each standards as standard (standard.name)}
			<li class="flex flex-wrap items-center justify-between gap-x-6 gap-y-1 px-6 py-4">
				<span class="flex items-center gap-3 text-lg font-semibold text-slate-900">
					<Check class="size-5 shrink-0 text-green-600" aria-hidden="true" />
					{standard.name}
				</span>
				<span class="text-slate-600">{standard.scope}</span>
			</li>
		{/each}
	</ul>
	<p class="mt-3 text-sm text-slate-600">
		Counts reflect the catalog shipped in Credimi on {fcafCatalog.asOf}. New suites plug into
		the same pipelines, results pages and evidence model.
	</p>
</div>
