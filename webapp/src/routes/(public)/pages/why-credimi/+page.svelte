<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import ArrowRight from '@lucide/svelte/icons/arrow-right';
	import BadgeCheck from '@lucide/svelte/icons/badge-check';
	import Check from '@lucide/svelte/icons/check';
	import PlayCircle from '@lucide/svelte/icons/circle-play';
	import Cloud from '@lucide/svelte/icons/cloud';
	import FileCheck from '@lucide/svelte/icons/file-check';
	import Smartphone from '@lucide/svelte/icons/smartphone';
	import {
		columns,
		competitors,
		eudiGaps,
		chooseCredimi,
		credimiBestAt,
		matrices,
		supportLabels,
		type ColumnId,
		type Support
	} from '$lib/content/why-credimi';
	import { entities, type EntityData } from '$lib/global/entities';
	import RoleTile, { type RoleStyle } from '$lib/layout/marketing/role-tile.svelte';
	import StandardsCoverage from '$lib/layout/marketing/standards-coverage.svelte';

	import type { IconComponent } from '@/components/types';

	import Button from '@/components/ui-custom/button.svelte';

	import CapabilityMatrix from './_partials/capability-matrix.svelte';
	import SupportDot from './_partials/support-dot.svelte';

	// Type roles: display, section heading, lead, body. One family (Inter), contrast from size and weight.
	const container = 'mx-auto max-w-screen-xl px-4 md:px-8';
	const h2 =
		'text-[clamp(2rem,3.6vw,3.25rem)] leading-[1.08] font-bold tracking-[-0.025em] text-balance';
	const lead = 'text-xl leading-relaxed text-pretty';
	const body = 'text-lg leading-[1.65] text-pretty';

	// Role colors and icons come from the Hub entities, so the page speaks the product's color code.
	const issuer = entities.credential_issuers;
	const wallet = entities.wallets;
	const verifier = entities.verifiers;

	// Credimi itself does the judging, so validation wears the brand indigo instead of a second red.
	const validation: RoleStyle = {
		icon: entities.conformance_checks.icon,
		classes: { bg: 'bg-secondary', text: 'text-primary', border: 'border-primary' }
	};

	const journey: { entity: EntityData; role: string; action: string; detail: string }[] = [
		{
			entity: issuer,
			role: 'Live issuer',
			action: 'sends a fresh credential offer',
			detail: 'openid-credential-offer://'
		},
		{
			entity: wallet,
			role: 'Your wallet',
			action: 'accepts the credential on a real phone',
			detail: 'SD-JWT VC · mDoc'
		},
		{
			entity: verifier,
			role: 'Live verifier',
			action: 'requests and checks a presentation',
			detail: 'OpenID4VP 1.0'
		}
	];

	const facts = [
		{ term: 'Tests', value: 'Wallets, credential issuers, verifiers' },
		{ term: 'Frameworks', value: 'EUDI FCAF, OpenID4VCI, OpenID4VP' },
		{ term: 'Runs on', value: 'Android devices, emulators, iOS simulators' },
		{ term: 'Deployment', value: 'credimi.io or self-hosted, AGPL-3.0' }
	];

	const pipelineSteps: { entity: RoleStyle; actor: string; text: string }[] = [
		{
			entity: issuer,
			actor: 'Issuer',
			text: 'Ask a live issuer for a fresh credential offer (StepCI).'
		},
		{
			entity: wallet,
			actor: 'Wallet',
			text: 'Open the deeplink in a wallet on a phone, emulator or iOS simulator, and accept the credential (Maestro).'
		},
		{
			entity: verifier,
			actor: 'Verifier',
			text: 'Ask a verifier for a presentation request and complete it in the same wallet.'
		},
		{
			entity: validation,
			actor: 'Validate',
			text: 'Judge what happened against FCAF and protocol test cases.'
		},
		{
			entity: entities.pipelines,
			actor: 'Evidence',
			text: 'Store every input, output and artifact as reviewable evidence.'
		}
	];

	const categoryIcons: Record<Exclude<ColumnId, 'credimi'>, IconComponent> = {
		deviceClouds: Smartphone,
		maestro: PlayCircle,
		labs: Cloud,
		suites: BadgeCheck
	};

	const competitorGroups = (['deviceClouds', 'maestro', 'labs', 'suites'] as const).map((id) => ({
		id,
		icon: categoryIcons[id],
		category: columns[id].category,
		gap: eudiGaps[id],
		products: competitors.filter((competitor) => competitor.column === id)
	}));

	const legend: Support[] = ['core', 'partial', 'none'];

	const hubFeatures: { entity: EntityData; text: string }[] = [
		{
			entity: entities.credentials,
			text: 'Open a credential page and get a fresh credential offer as a QR code or deeplink.'
		},
		{
			entity: entities.use_cases_verifications,
			text: 'Open a verification use case and get a fresh presentation request.'
		},
		{
			entity: entities.pipelines,
			text: 'Reuse the same entry, without changes, as a step in your automated pipelines.'
		}
	];

	const related = [
		{
			kind: 'Atlas',
			title: 'EUDI Trust & Conformance Atlas: the Wallet rulebook, mapped',
			href: 'https://atlas.credimi.io/',
			action: 'Open the atlas ↗'
		},
		{
			kind: 'Article',
			title: 'What is conformance automation',
			href: '/pages/conformance-automation',
			action: 'Read article ↗'
		},
		{
			kind: 'Docs',
			title: 'Build and run wallet × issuer × verifier pipelines',
			href: 'https://docs.credimi.io/manual/create-first-pipeline/',
			action: 'Read docs ↗'
		}
	];

	function productNames(id: ColumnId) {
		return columns[id].products.join(', ');
	}
</script>

{#snippet diagram(src: string, alt: string, height: number)}
	<figure class="mt-12 overflow-x-auto rounded-2xl border border-border bg-card">
		<img
			{src}
			{alt}
			width="1200"
			{height}
			loading="lazy"
			class="block h-auto w-full min-w-[56rem]"
		/>
	</figure>
{/snippet}

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
					Prove your EUDI wallet works with real issuers and verifiers.
				</h1>
				<p class="mt-7 max-w-[58ch] {lead} text-primary-foreground/85">
					Device clouds check that your app opens on a phone. Credimi sends a live
					credential from a real issuer to your wallet, presents it to a real verifier,
					and checks every step against FCAF and OpenID4VC.
				</p>
				<div class="mt-9 flex flex-wrap gap-3">
					<Button href="/hub" size="lg" variant="secondary">Explore the Hub</Button>
					<Button
						href="/my/tests/new"
						size="lg"
						variant="outline"
						class="border-primary-foreground/40 bg-transparent text-primary-foreground hover:bg-primary-foreground/10 hover:text-primary-foreground"
					>
						Start a new test
					</Button>
				</div>
			</div>

			<!-- The journey: what one Credimi run actually exercises -->
			<ol class="journey relative space-y-4" aria-label="One Credimi run">
				{#each journey as step, i (step.role)}
					<li
						class="relative flex items-center gap-4 rounded-2xl bg-card p-5 text-card-foreground"
					>
						<RoleTile role={step.entity} />
						<div class="min-w-0">
							<p class="text-lg leading-tight font-bold">
								{step.role}
								<span class="font-normal text-slate-600">{step.action}</span>
							</p>
							<p
								class={[
									'mt-1 truncate font-mono text-sm',
									step.entity.classes.text
								]}
							>
								{step.detail}
							</p>
						</div>
						{#if i < journey.length - 1}
							<span class="journey-link" aria-hidden="true"></span>
						{/if}
					</li>
				{/each}
				<li
					class="flex items-center gap-3 rounded-2xl border border-primary-foreground/25 px-5 py-4"
				>
					<FileCheck class="size-6 shrink-0 text-green-400" aria-hidden="true" />
					<p class="font-semibold">
						Judged against FCAF and protocol test cases, with evidence
					</p>
				</li>
			</ol>
		</div>

		<dl
			class="mt-16 grid divide-primary-foreground/15 border-t border-primary-foreground/15 sm:grid-cols-2 lg:grid-cols-4 lg:divide-x"
		>
			{#each facts as fact (fact.term)}
				<div class="py-5 lg:px-6 lg:first:pl-0">
					<dt class="text-sm text-primary-foreground/70">{fact.term}</dt>
					<dd class="mt-1 font-semibold">{fact.value}</dd>
				</div>
			{/each}
		</dl>
	</div>
</section>

<!-- Two questions -->
<section class="bg-background">
	<div class="{container} py-20 md:py-28">
		<div class="grid gap-5 md:grid-cols-2">
			<div class="rounded-2xl bg-slate-100 p-8 md:p-10">
				<div class="flex gap-2 text-slate-400" aria-hidden="true">
					{#each { length: 6 }, i (i)}
						<Smartphone class="size-7" />
					{/each}
				</div>
				<p class="mt-8 text-lg text-slate-600">A device cloud asks:</p>
				<p
					class="mt-2 text-[clamp(1.75rem,2.6vw,2.25rem)] leading-tight font-bold tracking-[-0.02em] text-balance text-slate-800"
				>
					Does my app work on this device?
				</p>
			</div>
			<div class="rounded-2xl bg-secondary p-8 md:p-10">
				<div class="flex items-center gap-2" aria-hidden="true">
					<RoleTile role={issuer} size="sm" />
					<ArrowRight class="size-4 text-primary/50" />
					<RoleTile role={wallet} size="sm" />
					<ArrowRight class="size-4 text-primary/50" />
					<RoleTile role={verifier} size="sm" />
				</div>
				<p class="mt-8 text-lg text-primary/80">Credimi asks:</p>
				<p
					class="mt-2 text-[clamp(1.75rem,2.6vw,2.25rem)] leading-tight font-bold tracking-[-0.02em] text-balance text-primary"
				>
					Does my wallet, issuer or verifier conform, and does it work with the rest of
					the EUDI ecosystem?
				</p>
			</div>
		</div>
		<p class="mt-12 max-w-[68ch] {lead} text-slate-700">
			A European Digital Identity Wallet is never tested alone. It succeeds only when a
			credential issuer, a wallet and a verifier complete a flow together, under OpenID4VCI,
			OpenID4VP, SD-JWT VC, mDoc and the EUDI ARF, with evidence an assessor can trust.
		</p>
	</div>
</section>

<!-- How a pipeline works -->
<section class="bg-secondary">
	<div class="{container} py-20 md:py-28">
		<h2 class="{h2} max-w-[20ch] text-primary">
			One pipeline tests the flow between components
		</h2>
		<p class="mt-6 max-w-[60ch] {lead} text-slate-700">
			Most testing platforms see one system under test. Credimi drives every party of the
			flow, in order, and keeps the record.
		</p>

		<div class="mt-12 overflow-hidden rounded-2xl border border-border bg-card">
			<div
				class="flex flex-wrap items-center justify-between gap-2 bg-primary px-6 py-4 text-primary-foreground"
			>
				<p class="font-bold">Credimi pipeline</p>
				<p class="font-mono text-sm text-primary-foreground/75">
					orchestrated and timestamped by Temporal
				</p>
			</div>
			<ol class="grid divide-y divide-border lg:grid-cols-5 lg:divide-x lg:divide-y-0">
				{#each pipelineSteps as step, i (step.actor)}
					<li class="relative p-6">
						<div class="flex items-center justify-between">
							<RoleTile role={step.entity} />
							<span class="font-mono text-sm text-slate-500">{i + 1} / 5</span>
						</div>
						<p class={['mt-5 text-xl font-bold', step.entity.classes.text]}>
							{step.actor}
						</p>
						<p class="mt-2 leading-relaxed text-slate-700">{step.text}</p>
					</li>
				{/each}
			</ol>
		</div>

		<blockquote
			class="mt-16 max-w-[34ch] text-[clamp(1.6rem,2.6vw,2.25rem)] leading-tight font-bold tracking-[-0.02em] text-balance text-primary"
		>
			A device cloud can run one step of an EUDI flow. Credimi runs the whole journey and
			judges it.
		</blockquote>
	</div>
</section>

<!-- What each competitor is best at -->
<section class="bg-background">
	<div class="{container} py-20 md:py-28">
		<h2 class="{h2} max-w-[20ch] text-primary">Each tool is good at its own job</h2>
		<p class="mt-6 max-w-[60ch] {lead} text-slate-700">
			Credimi works alongside these products. Each one answers its own question well; none was
			built to test EUDI wallets, issuers and verifiers together.
		</p>

		<div class="mt-12 space-y-5">
			{#each competitorGroups as group (group.id)}
				{@const Icon = group.icon}
				<article
					class="grid overflow-hidden rounded-2xl border border-border bg-card lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]"
				>
					<div class="bg-amber-50 p-6 lg:p-8">
						<div class="flex items-center gap-3">
							<span
								class="grid size-11 place-items-center rounded-xl bg-white text-slate-700 ring-1 ring-amber-200"
							>
								<Icon class="size-5" aria-hidden="true" />
							</span>
							<h3 class="text-xl font-bold text-slate-900">{group.category}</h3>
						</div>
						<p class="mt-6 text-sm font-semibold text-amber-800">For EUDIW testing:</p>
						<p class="mt-1 leading-relaxed text-pretty text-slate-800">{group.gap}</p>
					</div>
					<ul class="divide-y divide-border">
						{#each group.products as product (product.name)}
							<li
								class="grid gap-2 p-6 sm:grid-cols-[13rem_minmax(0,1fr)] sm:gap-6 lg:px-8"
							>
								<p class="text-lg leading-snug font-bold text-slate-900">
									{product.name}
								</p>
								<p
									class="flex gap-3 text-[1.0625rem] leading-relaxed text-pretty text-slate-700"
								>
									<Check
										class="mt-1 size-5 shrink-0 text-green-600"
										aria-hidden="true"
									/>
									<span>
										<span class="font-semibold text-slate-900">Best at:</span>
										{product.bestAt}
									</span>
								</p>
							</li>
						{/each}
					</ul>
				</article>
			{/each}

			<article
				class="fold-motif grid rounded-2xl bg-primary text-primary-foreground lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]"
			>
				<div
					class="relative border-b border-primary-foreground/15 p-6 lg:border-r lg:border-b-0 lg:p-8"
				>
					<div class="flex items-center gap-2" aria-hidden="true">
						<RoleTile role={issuer} size="sm" />
						<RoleTile role={wallet} size="sm" />
						<RoleTile role={verifier} size="sm" />
					</div>
					<h3 class="mt-5 text-xl font-bold">Credimi</h3>
					<p class="mt-2 leading-relaxed text-primary-foreground/85">
						Built for the EUDI question. It runs the suites above for you and adds real
						wallets on real devices.
					</p>
				</div>
				<ul class="relative divide-y divide-primary-foreground/15">
					{#each credimiBestAt as item (item)}
						<li class="flex gap-3 p-6 text-lg leading-relaxed lg:px-8">
							<Check
								class="mt-1.5 size-5 shrink-0 text-green-400"
								aria-hidden="true"
							/>
							<span><span class="font-semibold">Best at:</span> {item}</span>
						</li>
					{/each}
				</ul>
			</article>
		</div>
	</div>
</section>

<!-- Capability matrices -->
<section class="bg-secondary">
	<div class="{container} py-20 md:py-28">
		<h2 class="{h2} text-primary">Compare capabilities:</h2>
		<p class="mt-6 max-w-[60ch] {lead} text-slate-700">
			Grouped by product category. Each column names the products it covers.
		</p>
		<ul
			class="mt-8 inline-flex flex-wrap gap-x-8 gap-y-3 rounded-xl bg-card px-5 py-3"
			aria-label="Legend"
		>
			{#each legend as level (level)}
				<li class="flex items-center gap-2.5 text-slate-700">
					<SupportDot value={level} />
					<span aria-hidden="true">{supportLabels[level]}</span>
				</li>
			{/each}
		</ul>

		<div class="mt-12 space-y-16">
			{#each matrices as matrix (matrix.id)}
				<CapabilityMatrix {matrix} />
			{/each}
		</div>
	</div>
</section>

<!-- Standards -->
<section class="bg-background">
	<div class="{container} py-20 md:py-28">
		<div class="grid gap-12 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] lg:gap-16">
			<div>
				<h2 class="{h2} text-primary">Standards and conformance coverage:</h2>
				<p class="mt-6 {lead} text-slate-700">
					Credimi does not replace the reference suites. It runs them for you, side by
					side, and adds what they lack: real wallets on real devices.
				</p>
				<p class="mt-5 {body} text-slate-600">
					{productNames('deviceClouds')} and {productNames('maestro')} cover none of these.
				</p>
			</div>

			<StandardsCoverage />
		</div>

		<aside
			class="mt-16 flex flex-wrap items-center justify-between gap-6 rounded-2xl bg-secondary p-8 md:p-10"
		>
			<div class="max-w-[52ch]">
				<p class="text-2xl leading-snug font-bold text-balance text-primary">
					Get a live credential offer into your wallet in under a minute.
				</p>
				<p class="mt-2 {body} text-slate-700">
					Every issuer and verifier on the Hub exposes a working flow. No account needed.
				</p>
			</div>
			<Button href="/hub" size="lg">Explore the Hub</Button>
		</aside>
	</div>
</section>

<!-- FCAF -->
<section class="bg-secondary">
	<div class="{container} py-20 md:py-28">
		<div class="grid gap-10 lg:grid-cols-2 lg:gap-16">
			<h2 class="{h2} text-primary">FCAF test cases that come with their evidence</h2>
			<div class="space-y-5 {body} text-slate-700">
				<p>
					The EUDI Functional Conformance Assessment Framework (FCAF) describes what an
					EUDI wallet solution must do. Credimi runs FCAF test cases inside its pipelines:
				</p>
				<ul class="list-disc space-y-2 pl-6 marker:text-primary">
					<li>
						every scenario produces named evidence: credential offers, presentation
						requests, responses, screenshots;
					</li>
					<li>
						one final validation step judges all catalog test cases against that
						evidence;
					</li>
					<li>
						the full report is stored with the run, and each evidence item gets a
						SHA-256 hash and a pointer into the run's Temporal history.
					</li>
				</ul>
			</div>
		</div>

		{@render diagram(
			'/pages/why-credimi/fcaf-report.svg',
			'Sample FCAF report: each test case links to hashed evidence in the run history',
			460
		)}

		<p
			class="mt-10 max-w-[44ch] text-2xl leading-snug font-bold tracking-[-0.02em] text-balance text-primary"
		>
			Hand the report to an assessor: every claim in it points back to what actually happened.
		</p>
	</div>
</section>

<!-- Runners -->
<section class="bg-background">
	<div class="{container} py-20 md:py-28">
		<div class="grid gap-10 lg:grid-cols-2 lg:gap-16">
			<h2 class="{h2} text-primary">Keep your devices, network and data</h2>
			<div class="space-y-5 {body} text-slate-700">
				<p>
					Identity test data is sensitive, and many issuers and verifiers live behind VPNs
					or in pre-production networks.
				</p>
				<ul class="list-disc space-y-2 pl-6 marker:text-primary">
					<li>
						Credimi runners connect your own Android phones, emulators, Redroid
						instances or iOS simulators to the platform.
					</li>
					<li>
						Runners can be private to your organization or published for others to use.
					</li>
					<li>Each organization runs in its own isolated Temporal namespace.</li>
					<li>
						Credimi is open source (AGPL-3.0), so you can inspect it or host it
						yourself.
					</li>
				</ul>
			</div>
		</div>

		{@render diagram(
			'/pages/why-credimi/runners.svg',
			'Your own devices, emulators and simulators connect to Credimi through a credimi-runner',
			400
		)}

		<p class="mt-10 max-w-[68ch] {body} text-slate-700">
			Device clouds run on their hardware, in their regions. That suits generic apps. It is a
			harder conversation when the payload is a Person Identification Data credential.
		</p>
	</div>
</section>

<!-- Hub -->
<section class="bg-secondary">
	<div class="{container} py-20 md:py-28">
		<div class="grid gap-10 lg:grid-cols-2 lg:gap-16">
			<div>
				<h2 class="{h2} text-primary">Test against the real ecosystem, not mocks</h2>
				<p class="mt-6 {body} text-slate-700">
					Every issuer, verifier, wallet and credential published on the
					<a href="/hub" class="font-medium text-primary underline underline-offset-4"
						>Credimi Hub</a
					>
					comes with a live integration. The Hub is an interoperability network, not a test
					lab.
				</p>
			</div>
			<ul class="space-y-3">
				{#each hubFeatures as feature (feature.text)}
					<li class="flex items-center gap-4 rounded-2xl bg-card p-5">
						<RoleTile role={feature.entity} />
						<span class="{body} text-slate-800">{feature.text}</span>
					</li>
				{/each}
			</ul>
		</div>

		{@render diagram(
			'/pages/why-credimi/hub.svg',
			'The Credimi Hub connects published issuers, wallets and verifiers',
			420
		)}

		<blockquote
			class="mt-12 max-w-[34ch] text-[clamp(1.6rem,2.6vw,2.25rem)] leading-tight font-bold tracking-[-0.02em] text-balance text-primary"
		>
			Publish on the Hub, and every wallet team in the ecosystem can test against you.
		</blockquote>
	</div>
</section>

<!-- Choose Credimi + CTA -->
<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative py-20 md:py-28">
		<div class="grid gap-12 lg:grid-cols-2 lg:gap-16">
			<div>
				<h2 class={h2}>Choose Credimi when:</h2>
				<div class="mt-10 flex flex-wrap gap-3">
					<Button href="/my/tests/new" size="lg" variant="secondary"
						>Start a new test</Button
					>
					<a
						href="mailto:credimi@forkbomb.eu"
						class="inline-flex h-10 items-center rounded-md border border-primary-foreground/40 px-6 text-sm font-medium transition-colors duration-150 hover:bg-primary-foreground/10"
					>
						Talk to us
					</a>
				</div>
			</div>
			<ul class="divide-y divide-primary-foreground/15 border-y border-primary-foreground/15">
				{#each chooseCredimi as reason (reason)}
					<li class="flex gap-4 py-5 text-lg leading-relaxed text-pretty">
						<Check class="mt-1.5 size-5 shrink-0 text-green-400" aria-hidden="true" />
						<span>{reason}</span>
					</li>
				{/each}
			</ul>
		</div>
	</div>
</section>

<!-- Related -->
<section class="bg-secondary">
	<div class="{container} py-16">
		<h2 class="border-b border-border pb-3 text-2xl font-bold text-primary">
			Related resources:
		</h2>
		<div class="mt-6 grid gap-4 md:grid-cols-3">
			{#each related as item (item.href)}
				<a
					href={item.href}
					class="flex flex-col gap-2 rounded-2xl border border-border bg-card p-6 transition-colors duration-150 hover:border-primary"
				>
					<span class="text-sm font-medium text-slate-600">{item.kind}</span>
					<strong class="flex-1 text-lg leading-snug font-semibold text-slate-900"
						>{item.title}</strong
					>
					<span class="font-medium text-primary">{item.action}</span>
				</a>
			{/each}
		</div>
		<p class="mt-10 max-w-[90ch] text-sm leading-relaxed text-slate-600">
			Competitor capabilities are summarized from public product pages as of 20/10/2026 and
			grouped by product category. Product names are trademarks of their owners.
		</p>
	</div>
</section>

<style>
	/* The one authored motion: a signal travelling down the issuer → wallet → verifier links. */
	.journey-link {
		position: absolute;
		top: 100%;
		left: 2.75rem;
		width: 2px;
		height: 1rem;
		overflow: hidden;
		background: oklch(1 0 0 / 0.3);
	}

	.journey-link::after {
		position: absolute;
		inset: 0;
		background: var(--color-green-400);
		transform: translateY(-100%);
		animation: signal 1.8s cubic-bezier(0.16, 1, 0.3, 1) infinite;
		content: '';
	}

	.journey li:nth-child(2) .journey-link::after {
		animation-delay: 0.6s;
	}

	@keyframes signal {
		to {
			transform: translateY(100%);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.journey-link::after {
			animation: none;
			transform: none;
		}
	}
</style>
