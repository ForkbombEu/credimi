<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import FileCheck from '@lucide/svelte/icons/file-check';
	import { entities } from '$lib/global/index.js';
	import { getHubItemLogo, getHubItemUrl, HubItemCard, type HubItem } from '$lib/hub';
	import RoleTile, { type RoleStyle } from '$lib/layout/marketing/role-tile.svelte';
	import { fcafCatalog } from '$lib/layout/marketing/standards';

	import Button from '@/components/ui-custom/button.svelte';
	import { Badge } from '@/components/ui/badge';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import ScoreboardSection from './_partials/scoreboard-section.svelte';

	let { data } = $props();
	const { wallets, issuers, verifiers, scoreboard, counts } = $derived(data);

	const container = 'mx-auto max-w-screen-xl px-4 md:px-8';
	const h2 =
		'text-[clamp(1.75rem,3.2vw,2.75rem)] leading-[1.1] font-bold tracking-[-0.025em] text-balance';

	// One live example per role, from the random Hub samples; role labels when the Hub is empty.
	const journey = $derived([
		{ role: entities.credential_issuers, item: issuers[0], action: m.home_issuer_action() },
		{ role: entities.wallets, item: wallets[0], action: m.home_wallet_action() },
		{ role: entities.verifiers, item: verifiers[0], action: m.home_verifier_action() }
	]);

	const facts = $derived([
		{ value: counts.wallets, label: entities.wallets.labels.plural },
		{ value: counts.issuers, label: entities.credential_issuers.labels.plural },
		{ value: counts.verifiers, label: entities.verifiers.labels.plural }
	]);

	const validation: RoleStyle = {
		icon: entities.conformance_checks.icon,
		classes: { bg: 'bg-secondary', text: 'text-primary', border: 'border-primary' }
	};

	const paths: { role: RoleStyle; title: string; text: string; href: string }[] = [
		{
			role: validation,
			title: m.home_path_conformance_title(),
			text: m.home_path_conformance_text(),
			href: '/pages/conformance-automation'
		},
		{
			role: entities.pipelines,
			title: m.home_path_interop_title(),
			text: m.home_path_interop_text(),
			href: '/pages/why-credimi'
		},
		{
			role: {
				icon: entities.scoreboard.icon,
				classes: { bg: 'bg-green-100', text: 'text-green-700', border: 'border-green-700' }
			},
			title: m.home_path_compare_title(),
			text: m.home_path_compare_text(),
			href: '/scoreboard'
		}
	];

	const rows: { role: RoleStyle & { labels: { plural?: string } }; items: HubItem[] }[] =
		$derived([
			{ role: entities.credential_issuers, items: issuers },
			{ role: entities.verifiers, items: verifiers },
			{ role: entities.wallets, items: wallets }
		]);
</script>

<!-- Hero -->
<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative pt-16 pb-12 md:pt-24 md:pb-16">
		<div
			class="grid grid-cols-[minmax(0,1fr)] items-center gap-14 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)] lg:gap-16"
		>
			<div>
				<h1
					class="text-[clamp(2.5rem,5.2vw,4.5rem)] leading-[1.04] font-bold tracking-[-0.03em] text-balance"
				>
					{m.home_title()}
				</h1>
				<p
					class="mt-7 max-w-[58ch] text-xl leading-relaxed text-pretty text-primary-foreground/85"
				>
					{m.home_lead()}
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

			<div>
				<p class="mb-3 text-sm text-primary-foreground/75">{m.home_live_on_hub()}</p>
				<ol class="space-y-3">
					{#each journey as step (step.role.slug)}
						{@const logo = step.item ? getHubItemLogo(step.item) : undefined}
						<li>
							<svelte:element
								this={step.item ? 'a' : 'div'}
								href={step.item ? getHubItemUrl(step.item) : undefined}
								class="flex items-center gap-4 rounded-2xl bg-card p-4 text-card-foreground transition-colors duration-150 hover:bg-secondary"
							>
								{#if logo}
									<img
										src={logo}
										alt=""
										class="size-12 shrink-0 rounded-xl border border-border object-contain p-1"
									/>
								{:else}
									<RoleTile role={step.role} />
								{/if}
								<div class="min-w-0">
									<p class="truncate text-lg leading-tight font-bold">
										{step.item?.name ?? step.role.labels.singular}
									</p>
									<p class="mt-0.5 text-sm text-slate-600">
										<span class={['font-semibold', step.role.classes.text]}>
											{step.role.labels.singular}
										</span>
										{step.action}
									</p>
								</div>
							</svelte:element>
						</li>
					{/each}
					<li
						class="flex items-center gap-3 rounded-2xl border border-primary-foreground/25 px-5 py-4"
					>
						<FileCheck class="size-6 shrink-0 text-green-400" aria-hidden="true" />
						<p class="font-semibold">{m.home_judged()}</p>
					</li>
				</ol>
			</div>
		</div>

		<dl
			class="mt-16 grid grid-cols-2 divide-primary-foreground/15 border-t border-primary-foreground/15 lg:grid-cols-4 lg:divide-x"
		>
			{#each facts.filter((fact) => fact.value > 0) as fact (fact.label)}
				<div class="py-5 lg:px-6 lg:first:pl-0">
					<dt class="text-sm text-primary-foreground/70">
						{fact.label}
						{m.home_on_the_hub()}
					</dt>
					<dd class="mt-1 text-3xl font-bold tracking-[-0.02em]">{fact.value}</dd>
				</div>
			{/each}
			<div class="py-5 lg:px-6">
				<dt class="text-sm text-primary-foreground/70">{m.home_fcaf_cases()}</dt>
				<dd class="mt-1 text-3xl font-bold tracking-[-0.02em]">{fcafCatalog.testCases}</dd>
			</div>
		</dl>
	</div>
</section>

<!-- Paths -->
<section class="bg-background">
	<div class="{container} py-16 md:py-24">
		<h2 class="{h2} text-primary">{m.home_paths_title()}</h2>
		<ul class="mt-10 grid gap-4 md:grid-cols-3">
			{#each paths as path (path.href)}
				<li>
					<a
						href={path.href}
						class="group flex h-full flex-col gap-4 rounded-2xl border border-border bg-card p-6 transition-colors duration-150 hover:border-primary lg:p-7"
					>
						<RoleTile role={path.role} />
						<h3 class={['text-xl font-bold', path.role.classes.text]}>{path.title}</h3>
						<p class="flex-1 text-lg leading-relaxed text-pretty text-slate-700">
							{path.text}
						</p>
						<ArrowRight
							class="size-5 text-primary transition-transform duration-150 group-hover:translate-x-1"
							aria-hidden="true"
						/>
					</a>
				</li>
			{/each}
		</ul>
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
			{#each rows as row (row.role.labels.plural)}
				{#if row.items.length}
					<div>
						<div class="flex items-center gap-3">
							<RoleTile role={row.role} size="sm" />
							<h3 class="text-xl font-bold text-slate-900">
								{row.role.labels.plural}
							</h3>
						</div>
						<div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-3">
							{#each row.items as item (item.id)}
								<HubItemCard {item} />
							{/each}
						</div>
					</div>
				{/if}
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

<!-- CTA -->
<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative py-16 md:py-24">
		<div class="max-w-[48rem]">
			<h2 class={h2}>{m.home_cta_title()}</h2>
			<p class="mt-5 text-xl leading-relaxed text-pretty text-primary-foreground/85">
				{m.home_cta_lead()}
			</p>
			<div class="mt-8 flex flex-wrap gap-3">
				<Button href="/my/tests/new" size="lg" variant="secondary"
					>{m.Start_a_new_test()}</Button
				>
				<a
					href="https://docs.credimi.io/"
					class="inline-flex h-10 items-center rounded-md border border-primary-foreground/40 px-6 text-sm font-medium transition-colors duration-150 hover:bg-primary-foreground/10"
				>
					{m.home_read_docs()}
				</a>
			</div>
		</div>
	</div>
</section>
