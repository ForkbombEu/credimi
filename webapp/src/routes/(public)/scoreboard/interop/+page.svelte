<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { InteropMode, InteropStatus } from '$lib/scoreboard/interop/types';

	import { resolve } from '$app/paths';
	import MatrixGrid from '$lib/scoreboard/interop/matrix-grid.svelte';
	import { interopStatusStyles } from '$lib/scoreboard/interop/status';

	import PublicPageHeader from '@/components/layout/public-page-header.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { localizeHref, m } from '@/i18n';

	let { data } = $props();

	const modeTabs: { value: InteropMode; label: () => string }[] = [
		{ value: 'wallets_credentials', label: () => m.interop_mode_wallets_credentials() },
		{ value: 'wallets_issuers', label: () => m.interop_mode_wallets_issuers() },
		{ value: 'wallets_verifiers', label: () => m.interop_mode_wallets_verifiers() },
		{
			value: 'wallets_use_case_verifications',
			label: () => m.interop_mode_wallets_use_case_verifications()
		},
		{
			value: 'wallets_conformance_checks',
			label: () => m.interop_mode_wallets_conformance_checks()
		},
		{
			value: 'use_case_verifications_conformance_checks',
			label: () => m.interop_mode_use_case_verifications_conformance_checks()
		}
	];

	const legendItems: { status: InteropStatus; label: () => string }[] = [
		{ status: 'broken', label: () => m.interop_matrix_legend_broken() },
		{ status: 'failing', label: () => m.interop_matrix_legend_failing() },
		{ status: 'flaky', label: () => m.interop_matrix_legend_flaky() },
		{ status: 'stable', label: () => m.interop_matrix_legend_stable() }
	];
</script>

<div class="grow bg-secondary pt-0 pb-20">
	<PublicPageHeader entity="scoreboard" description={m.interop_matrix_description()} />
	<div class="mx-auto mb-6 flex max-w-7xl justify-center px-4 md:px-8">
		<Button variant="outline" href="/scoreboard">{m.Back()}</Button>
	</div>
	<div class="mx-auto mb-4 flex max-w-7xl justify-center px-4 md:px-8">
		<div class="inline-flex rounded-md border bg-background p-1">
			{#each modeTabs as tab (tab.value)}
				<a
					class={`rounded-sm px-3 py-1.5 text-sm ${
						data.mode === tab.value
							? 'bg-primary text-primary-foreground'
							: 'text-muted-foreground hover:text-foreground'
					}`}
					href={resolve(localizeHref(`/scoreboard/interop?mode=${tab.value}`) as '/')}
					aria-current={data.mode === tab.value ? 'page' : undefined}
				>
					{tab.label()}
				</a>
			{/each}
		</div>
	</div>
	<MatrixGrid matrix={data.matrix}>
		{#snippet legend()}
			<span class="text-xs font-semibold tracking-wide text-muted-foreground uppercase"
				>Cross</span
			>
			{#each legendItems as item (item.status)}
				{@const styles = interopStatusStyles(item.status)}
				<span class="inline-flex items-center gap-1.5 text-sm">
					<span class="size-2.5 rounded-full {styles.dot}"></span>
					{item.label()}
				</span>
			{/each}
		{/snippet}
	</MatrixGrid>
	<p class="mx-auto mt-6 max-w-7xl px-4 text-sm text-muted-foreground md:px-8">
		{m.interop_matrix_footnote()}
	</p>
</div>
