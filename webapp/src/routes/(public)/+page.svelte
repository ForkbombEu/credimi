<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import FileCheck from '@lucide/svelte/icons/file-check';
	import {
		credentialIssuersAndCredentialsSection,
		entities,
		verifiersAndUseCaseVerificationsSection
	} from '$lib/global/index.js';
	import { HubItemCard } from '$lib/hub';
	import RoleTile from '$lib/layout/marketing/role-tile.svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import ScoreboardSection from './_partials/scoreboard-section.svelte';

	let { data } = $props();
	const { wallets, issuers, verifiers, scoreboard } = $derived(data);

	const container = 'mx-auto max-w-screen-xl px-4 md:px-8';
	const h2 =
		'text-[clamp(1.75rem,3.2vw,2.75rem)] leading-[1.1] font-bold tracking-[-0.025em] text-balance';

	// `tab` is the /hub tab that lists the whole collection.
	const rows = $derived([
		{
			role: entities.credential_issuers,
			tab: credentialIssuersAndCredentialsSection.slug,
			list: issuers
		},
		{
			role: entities.verifiers,
			tab: verifiersAndUseCaseVerificationsSection.slug,
			list: verifiers
		},
		{ role: entities.wallets, tab: entities.wallets.slug, list: wallets }
	]);

	const moreSections = [entities.custom_checks, entities.pipelines, entities.conformance_checks];
</script>

<!-- Hero -->
<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative pt-16 pb-12 md:pt-24 md:pb-16">
		<div>
			<!-- Own, wider measure: at the body width the title broke into three lines with a lone "and Hub". -->
			<h1
				class="max-w-[62rem] text-[clamp(2.25rem,4.6vw,4rem)] leading-[1.05] font-bold tracking-[-0.03em] text-balance"
			>
				{m.EUDIW_Conformance_Interoperability_and_Hub()}
			</h1>
			<div
				class="mt-7 max-w-[46rem] space-y-2 text-xl leading-relaxed text-pretty text-primary-foreground/85"
			>
				<p>{m.Explore_the_hub_and_try_credentials_wallets_and_services()}</p>
				<p>{m.Test_the_conformance_and_interoperability_of_your_EUDIW()}</p>
			</div>
			<p class="mt-5 flex items-center gap-3 font-semibold">
				<FileCheck class="size-5 shrink-0 text-green-400" aria-hidden="true" />
				{m.home_judged()}
			</p>
			<div class="mt-9 flex flex-wrap gap-3">
				<Button href="/hub" size="lg" variant="secondary">{m.Explore_Hub()}</Button>
				<Button
					href={$currentUser ? '/my/pipelines' : '/login'}
					size="lg"
					variant="outline"
					class="border-primary-foreground/40 bg-transparent whitespace-normal text-primary-foreground hover:bg-primary-foreground/10 hover:text-primary-foreground"
				>
					{m.automated_conformance_interop()}
					<Badge
						variant="outline"
						class="border-primary-foreground/50 text-xs text-primary-foreground"
					>
						{m.Beta()}
					</Badge>
				</Button>
			</div>
		</div>
	</div>
</section>

<!-- Hub -->
<section class="bg-secondary">
	<div class="{container} py-16 md:py-24">
		<div class="flex flex-wrap items-end justify-between gap-4 border-b border-border pb-4">
			<h2 class="{h2} text-primary">{m.Find_solutions()}</h2>
			<a href="/hub" class="font-medium text-primary underline underline-offset-4">
				{m.Explore_Hub()} ↗
			</a>
		</div>
		<div class="mt-10 space-y-10">
			{#each rows as row (row.tab)}
				{#if row.list.items.length}
					<div>
						<div class="flex items-center gap-3">
							<RoleTile role={row.role} size="sm" />
							<h3 class="text-xl font-bold text-slate-900">
								{row.role.labels.plural}
							</h3>
							<span class="font-mono text-sm text-muted-foreground">
								{row.list.totalItems}
							</span>
							<a
								href="/hub?tab={row.tab}"
								aria-label="{m.home_see_all()} {row.role.labels.plural}"
								class="ml-auto text-sm font-medium text-primary underline underline-offset-4"
							>
								{m.home_see_all()} ↗
							</a>
						</div>
						<div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-3">
							{#each row.list.items as item (item.id)}
								<HubItemCard {item} />
							{/each}
						</div>
					</div>
				{/if}
			{/each}
		</div>
		<div class="mt-12 flex flex-wrap items-center gap-3">
			<span class="font-semibold text-slate-900">{m.home_also_in_hub()}</span>
			{#each moreSections as section (section.slug)}
				<a
					href="/hub?tab={section.slug}"
					class="inline-flex items-center gap-2.5 rounded-md border border-border bg-card py-1.5 pr-4 pl-1.5 text-sm font-medium text-slate-900 transition-colors duration-150 ease-out hover:border-primary hover:text-primary"
				>
					<RoleTile role={section} size="sm" />
					{section.labels.plural} ↗
				</a>
			{/each}
		</div>
	</div>
</section>

<!-- Scoreboard -->
<section class="bg-background">
	<div class="{container} py-16 md:py-24">
		<div class="flex flex-wrap items-end justify-between gap-4 border-b border-border pb-4">
			<h2 class="{h2} text-primary">{m.Compare_by_test_results()}</h2>
			<a href="/scoreboard" class="font-medium text-primary underline underline-offset-4">
				{m.View_Scoreboard()} ↗
			</a>
		</div>
		<div class="mt-8">
			<ScoreboardSection {...scoreboard} />
		</div>
	</div>
</section>

<!-- CTA: light band, so the page does not run into the indigo footer -->
<section class="bg-secondary">
	<div class="{container} py-16 md:py-24">
		<div class="max-w-[48rem]">
			<h2 class="{h2} text-primary">{m.home_cta_title()}</h2>
			<p class="mt-5 text-xl leading-relaxed text-pretty text-slate-700">
				{m.home_cta_lead()}
			</p>
			<div class="mt-8 flex flex-wrap gap-3">
				<Button href={$currentUser ? '/my' : '/login'} size="lg">
					{$currentUser ? m.Go_to_Dashboard() : m.Log_in()}
				</Button>
				<Button href="/scoreboard" size="lg" variant="outline">
					{m.home_see_fcaf_reports()}
				</Button>
			</div>
		</div>
	</div>
</section>
