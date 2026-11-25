<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ListChecksIcon } from '@lucide/svelte';

	import type { OrganizationsResponse, WalletsResponse } from '@/pocketbase/types';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';

	import { getFcafCoverage, type FcafCoverage } from './api';
	import { walletPath } from './builds';
	import {
		coverageCounts,
		coverageLabel,
		coverageVariant,
		coverageVerdict,
		percent
	} from './fcaf';

	//

	type Props = {
		wallet: WalletsResponse;
		/** The wallet's organization, whose path names the wallet. */
		walletOwner: Pick<OrganizationsResponse, 'canonified_name'>;
	};

	let { wallet, walletOwner }: Props = $props();

	let open = $state(false);
	let coverage = $state<Promise<FcafCoverage>>();

	$effect(() => {
		if (!open) return;
		coverage = getFcafCoverage(walletPath(walletOwner, wallet)).then((res) => {
			if (res.isErr) throw new Error(res.error.message);
			return res.value;
		});
	});
</script>

<Sheet bind:open title={m.Fcaf_coverage()} class="!w-[min(100vw,32rem)]">
	{#snippet trigger({ sheetTriggerAttributes })}
		<Button variant="outline" size="sm" {...sheetTriggerAttributes}>
			<ListChecksIcon />
			{m.Fcaf_coverage()}
		</Button>
	{/snippet}

	{#snippet content()}
		{#await coverage}
			<div class="flex items-center justify-center gap-2 py-6">
				<Spinner size={16} />
				<T class="text-muted-foreground">{m.Loading()}</T>
			</div>
		{:then result}
			{#if result}
				{@const { total, written, incomplete, recorded, supported } = coverageCounts(
					result.roles
				)}
				<div class="space-y-3 text-sm">
					{#each result.warnings as warning (warning)}
						<Alert variant="warning">
							<p class="break-words">{warning}</p>
						</Alert>
					{/each}

					<div class="font-mono text-xs text-muted-foreground">
						<p>
							{m.Fcaf_coverage_written({
								written,
								total,
								percent: percent(written, total)
							})}
						</p>
						<p>
							{m.Fcaf_coverage_supported({
								supported,
								recorded,
								percent: percent(supported, recorded)
							})}
						</p>
						{#if written - incomplete > recorded}
							<p>
								{m.Fcaf_coverage_unrecorded_count({
									count: written - incomplete - recorded
								})}
							</p>
						{/if}
					</div>

					<ul class="space-y-2">
						{#each result.roles as role (role.role)}
							{@const verdict = coverageVerdict(role)}
							<li class="space-y-1 rounded-md border p-3">
								<div class="flex items-center justify-between gap-2">
									<span class="truncate font-mono text-xs">{role.role}</span>
									<Badge variant={coverageVariant(verdict)}>
										{coverageLabel(verdict)}
									</Badge>
								</div>
								{#if role.summary}
									<p class="text-xs text-muted-foreground">{role.summary}</p>
								{/if}
								{#if verdict === 'unsupported' || verdict === 'finding'}
									<p class="text-xs break-words whitespace-pre-wrap">
										{role.note}
									</p>
								{:else if verdict === 'passed' && role.note}
									<p class="text-xs break-words whitespace-pre-wrap">
										{m.Review_agent_note()}: {role.note}
									</p>
								{:else if verdict === 'incomplete' && role.missing_fixtures.length}
									<p class="font-mono text-xs break-all text-muted-foreground">
										{m.Fcaf_coverage_missing_fixtures({
											fixtures: role.missing_fixtures.join(', ')
										})}
									</p>
								{:else if verdict === 'stale'}
									<p class="text-xs text-muted-foreground">
										{m.Fcaf_coverage_stale_hint()}
									</p>
								{/if}
							</li>
						{/each}
					</ul>
				</div>
			{/if}
		{:catch error}
			<p class="text-sm text-destructive" role="alert">{error.message}</p>
		{/await}
	{/snippet}
</Sheet>
