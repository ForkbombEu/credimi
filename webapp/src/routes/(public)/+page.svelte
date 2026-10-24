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

	// Guide pages are written in English only.
	const guides = [
		{
			href: '/pages/eudiw-conformance',
			title: 'EUDIW conformance: certification, FCAF and how to prepare',
			text: 'What the certification rules require and where FCAF stands today.'
		},
		{
			href: '/pages/fcaf-conformance',
			title: 'FCAF conformance testing for EUDI wallets',
			text: 'Run the FCAF test cases on real devices, with hashed evidence.'
		},
		{
			href: '/pages/eudi-wallet-interoperability-testing',
			title: 'EUDI wallet interoperability testing',
			text: 'Find out which wallets, issuers and verifiers work together.'
		}
	];

	// Real screens of a completed FCAF run on credimi.io (demo organisation, 16/10/2026).
	const tour = [
		{
			src: '/screens/pipeline-builder.webp',
			width: 1600,
			height: 1000,
			title: m.home_tour_build_title(),
			text: m.home_tour_build_text(),
			alt: m.home_tour_build_alt()
		},
		{
			src: '/screens/device-run.webp',
			// Portrait phone screen: narrower, so it does not tower over its text.
			narrow: true,
			width: 1000,
			height: 1093,
			title: m.home_tour_run_title(),
			text: m.home_tour_run_text(),
			alt: m.home_tour_run_alt()
		},
		{
			src: '/screens/run-timeline.webp',
			width: 1600,
			height: 1000,
			title: m.home_tour_trace_title(),
			text: m.home_tour_trace_text(),
			alt: m.home_tour_trace_alt()
		},
		{
			src: '/screens/fcaf-report.webp',
			width: 1400,
			height: 1167,
			title: m.home_tour_judge_title(),
			text: m.home_tour_judge_text(),
			alt: m.home_tour_judge_alt(),
			// A full complete-validation run, failures included, as a buyer would receive it.
			link: {
				href: 'https://credimi.io/api/files/pipeline_results/lm9cvfbiltat7mp/fcaf_assessment_hiyu281nr4.pdf',
				label: m.home_tour_judge_pdf()
			}
		}
	];
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

<!-- Tour: the private run views, shown before sign-in -->
<section class="bg-background">
	<div class="{container} py-16 md:py-24">
		<div class="border-b border-border pb-4">
			<h2 class="{h2} text-primary">{m.home_tour_title()}</h2>
			<p class="mt-4 max-w-[48rem] text-lg leading-relaxed text-pretty text-slate-700">
				{m.home_tour_lead()}
			</p>
		</div>
		<ol class="mt-12 space-y-16 md:space-y-24">
			{#each tour as step, i (step.src)}
				<li class="grid items-center gap-8 lg:grid-cols-12 lg:gap-12">
					<div class={['lg:col-span-5', i % 2 === 1 && 'lg:order-2']}>
						<span class="font-mono text-sm text-muted-foreground">
							{String(i + 1).padStart(2, '0')}
						</span>
						<h3 class="mt-2 text-2xl leading-snug font-bold text-slate-900">
							{step.title}
						</h3>
						<p class="mt-3 text-lg leading-relaxed text-pretty text-slate-700">
							{step.text}
						</p>
						{#if 'link' in step}
							<a
								href={step.link.href}
								class="mt-4 inline-block font-medium text-primary underline underline-offset-4"
							>
								{step.link.label} ↗
							</a>
						{/if}
					</div>
					<img
						src={step.src}
						width={step.width}
						height={step.height}
						alt={step.alt}
						loading="lazy"
						decoding="async"
						class={[
							'h-auto w-full rounded-[10px] border border-border bg-card lg:col-span-7',
							'narrow' in step && 'mx-auto max-w-xl'
						]}
					/>
				</li>
			{/each}
		</ol>
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
		<p class="mt-4 max-w-[48rem] text-lg leading-relaxed text-pretty text-slate-700">
			{m.home_hub_lead()}
		</p>
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
		<p class="mt-4 max-w-[48rem] text-lg leading-relaxed text-pretty text-slate-700">
			{m.home_scoreboard_lead()}
		</p>
		<div class="mt-8">
			<ScoreboardSection {...scoreboard} />
		</div>
	</div>
</section>

<!-- Guides: the explanations behind the product, for visitors new to FCAF and EUDI conformance -->
<section class="bg-secondary">
	<div class="{container} py-16 md:py-24">
		<div class="flex flex-wrap items-end justify-between gap-4 border-b border-border pb-4">
			<h2 class="{h2} text-primary">{m.home_guides_title()}</h2>
			<a href="/pages/guides" class="font-medium text-primary underline underline-offset-4">
				{m.home_see_all()} ↗
			</a>
		</div>
		<div class="mt-8 grid gap-4 md:grid-cols-3">
			{#each guides as guide (guide.href)}
				<a
					href={guide.href}
					class="flex flex-col gap-2 rounded-2xl border border-border bg-card p-6 transition-colors duration-150 hover:border-primary"
				>
					<strong class="flex-1 text-lg leading-snug font-semibold text-slate-900">
						{guide.title}
					</strong>
					<span class="text-slate-700">{guide.text}</span>
				</a>
			{/each}
		</div>
	</div>
</section>

<!-- CTA: light band, so the page does not run into the indigo footer -->
<section class="bg-background">
	<div class="{container} grid gap-12 py-16 md:py-24 lg:grid-cols-2">
		<div>
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
		<div class="rounded-[10px] border border-border bg-secondary p-8">
			<h2 class="text-2xl font-bold text-slate-900">{m.home_oss_title()}</h2>
			<p class="mt-3 text-lg leading-relaxed text-pretty text-slate-700">
				{m.home_oss_text()}
			</p>
			<a
				href="https://github.com/forkbombeu/credimi"
				class="mt-6 inline-block font-medium text-primary underline underline-offset-4"
			>
				{m.home_oss_star()} ↗
			</a>
		</div>
	</div>
</section>
