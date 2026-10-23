<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BadgeCheck from '@lucide/svelte/icons/badge-check';
	import Check from '@lucide/svelte/icons/check';
	import FileCheck from '@lucide/svelte/icons/file-check';
	import FileText from '@lucide/svelte/icons/file-text';
	import GitCommit from '@lucide/svelte/icons/git-commit-horizontal';
	import GitPullRequest from '@lucide/svelte/icons/git-pull-request';
	import Layers from '@lucide/svelte/icons/layers';
	import Minus from '@lucide/svelte/icons/minus';
	import Network from '@lucide/svelte/icons/network';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import Rocket from '@lucide/svelte/icons/rocket';
	import {
		automated,
		benefits,
		ciEndpoints,
		closing,
		definition,
		manual,
		steps,
		type BenefitId,
		type StepId
	} from '$lib/content/conformance-automation';
	import { entities } from '$lib/global/entities';
	import RoleTile, { type RoleStyle } from '$lib/layout/marketing/role-tile.svelte';
	import StandardsCoverage from '$lib/layout/marketing/standards-coverage.svelte';

	import type { IconComponent } from '@/components/types';

	import Button from '@/components/ui-custom/button.svelte';

	// Same type roles as the other marketing pages: display, section heading, lead, body.
	const container = 'mx-auto max-w-screen-xl px-4 md:px-8';
	const h2 =
		'text-[clamp(2rem,3.6vw,3.25rem)] leading-[1.08] font-bold tracking-[-0.025em] text-balance';
	const lead = 'text-xl leading-relaxed text-pretty';
	const body = 'text-lg leading-[1.65] text-pretty';

	const conformanceChecksHref = '/hub?tab=conformance-checks';

	const change: RoleStyle = {
		icon: GitCommit,
		classes: { bg: 'bg-slate-100', text: 'text-slate-700', border: 'border-slate-700' }
	};
	const evidence: RoleStyle = {
		icon: FileCheck,
		classes: { bg: 'bg-green-100', text: 'text-green-700', border: 'border-green-700' }
	};
	const gate: RoleStyle = {
		icon: GitPullRequest,
		classes: { bg: 'bg-secondary', text: 'text-primary', border: 'border-primary' }
	};

	const loop: { role: RoleStyle; title: string; detail: string }[] = [
		{ role: change, title: 'You push a change', detail: 'git push · new wallet APK' },
		{
			role: entities.conformance_checks,
			title: 'Credimi runs the suites',
			detail: 'OpenID4VCI · OpenID4VP · FCAF'
		},
		{
			role: evidence,
			title: 'The run keeps its evidence',
			detail: 'Temporal history · video · logs'
		},
		{
			role: gate,
			title: 'Your CI passes or blocks the release',
			detail: 'run-wallet-apk · run-issuer · run-verifier'
		}
	];

	const stepRoles: Record<StepId, RoleStyle> = {
		checks: entities.conformance_checks,
		pipeline: entities.pipelines,
		trigger: gate,
		evidence
	};

	const benefitIcons: Record<BenefitId, IconComponent> = {
		speed: Rocket,
		interop: Network,
		reports: FileText,
		standards: Layers
	};

	const related = [
		{
			kind: 'Comparison',
			title: 'Why Credimi',
			href: '/pages/why-credimi',
			action: 'Read comparison ↗'
		},
		{
			kind: 'Atlas',
			title: 'EUDI Trust & Conformance Atlas: the Wallet rulebook, mapped',
			href: 'https://atlas.credimi.io/',
			action: 'Open the atlas ↗'
		},
		{
			kind: 'Docs',
			title: 'Build and run wallet × issuer × verifier pipelines',
			href: 'https://docs.credimi.io/manual/testing-automation/build-and-run-pipelines/',
			action: 'Read docs ↗'
		}
	];
</script>

<!-- Hero -->
<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative pt-16 pb-16 md:pt-24 md:pb-24">
		<div
			class="grid grid-cols-[minmax(0,1fr)] items-center gap-14 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)] lg:gap-16"
		>
			<div>
				<h1
					class="text-[clamp(2.5rem,5.2vw,4.5rem)] leading-[1.04] font-bold tracking-[-0.03em] text-balance"
				>
					Check every release against the standards, automatically.
				</h1>
				<p class="mt-7 max-w-[58ch] {lead} text-primary-foreground/85">
					Conformance automation runs standard test suites against your wallet, credential
					issuer or verifier every time something changes, not once before an audit.
					Credimi runs those suites for you, on real devices, and keeps the evidence.
				</p>
				<div class="mt-9 flex flex-wrap gap-3">
					<Button href={conformanceChecksHref} size="lg" variant="secondary">
						Explore conformance checks
					</Button>
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

			<!-- The loop: what happens on every change -->
			<ol class="space-y-3" aria-label="What happens on every change">
				{#each loop as step (step.title)}
					<li
						class="flex items-center gap-4 rounded-2xl bg-card p-5 text-card-foreground"
					>
						<RoleTile role={step.role} />
						<div class="min-w-0">
							<p class="text-lg leading-tight font-bold">{step.title}</p>
							<p class={['mt-1 truncate font-mono text-sm', step.role.classes.text]}>
								{step.detail}
							</p>
						</div>
					</li>
				{/each}
				<li
					class="flex items-center gap-3 rounded-2xl border border-primary-foreground/25 px-5 py-4"
				>
					<RefreshCw class="size-6 shrink-0 text-green-400" aria-hidden="true" />
					<p class="font-semibold">Repeat on every change, or on a schedule</p>
				</li>
			</ol>
		</div>
	</div>
</section>

<!-- Definition and contrast -->
<section class="bg-background">
	<div class="{container} py-20 md:py-28">
		<div class="grid gap-10 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] lg:gap-16">
			<h2 class="{h2} text-primary">What is conformance automation?</h2>
			<div class="space-y-5 {lead} text-slate-700">
				{#each definition as paragraph (paragraph)}
					<p>{paragraph}</p>
				{/each}
			</div>
		</div>

		<div class="mt-14 grid gap-5 md:grid-cols-2">
			<div class="rounded-2xl bg-slate-100 p-8 md:p-10">
				<h3 class="text-2xl font-bold tracking-[-0.02em] text-slate-800">Manual review</h3>
				<ul class="mt-6 space-y-4">
					{#each manual as item (item)}
						<li class="flex gap-3 {body} text-slate-700">
							<Minus
								class="mt-1.5 size-5 shrink-0 text-slate-400"
								aria-hidden="true"
							/>
							<span>{item}</span>
						</li>
					{/each}
				</ul>
			</div>
			<div class="rounded-2xl bg-secondary p-8 md:p-10">
				<h3 class="text-2xl font-bold tracking-[-0.02em] text-primary">
					Automated conformance
				</h3>
				<ul class="mt-6 space-y-4">
					{#each automated as item (item)}
						<li class="flex gap-3 {body} text-slate-800">
							<Check
								class="mt-1.5 size-5 shrink-0 text-green-600"
								aria-hidden="true"
							/>
							<span>{item}</span>
						</li>
					{/each}
				</ul>
			</div>
		</div>
	</div>
</section>

<!-- How Credimi automates it -->
<section class="bg-secondary">
	<div class="{container} py-20 md:py-28">
		<h2 class="{h2} max-w-[20ch] text-primary">How Credimi automates it</h2>
		<p class="mt-6 max-w-[60ch] {lead} text-slate-700">
			Four steps, set up once. After that, every change runs through them on its own.
		</p>

		<ol
			class="mt-12 grid divide-y divide-border overflow-hidden rounded-2xl border border-border bg-card lg:grid-cols-4 lg:divide-x lg:divide-y-0"
		>
			{#each steps as step, i (step.title)}
				<li class="p-6 lg:p-7">
					<div class="flex items-center justify-between">
						<RoleTile role={stepRoles[step.id]} />
						<span class="font-mono text-sm text-slate-500"
							>{i + 1} / {steps.length}</span
						>
					</div>
					<p class={['mt-5 text-xl font-bold', stepRoles[step.id].classes.text]}>
						{step.title}
					</p>
					<p class="mt-2 leading-relaxed text-pretty text-slate-700">{step.text}</p>
				</li>
			{/each}
		</ol>

		<div
			class="mt-6 flex flex-wrap items-center gap-x-6 gap-y-2 rounded-2xl bg-card px-6 py-4 text-slate-700"
		>
			<span class="font-semibold text-slate-900">CI endpoints per role:</span>
			{#each ciEndpoints as endpoint (endpoint)}
				<code class="font-mono text-sm text-primary">{endpoint}</code>
			{/each}
		</div>
	</div>
</section>

<!-- Benefits -->
<section class="bg-background">
	<div class="{container} py-20 md:py-28">
		<h2 class="{h2} max-w-[20ch] text-primary">What changes for your team</h2>

		<ul class="mt-12 grid gap-x-16 md:grid-cols-2">
			{#each benefits as benefit (benefit.title)}
				{@const Icon = benefitIcons[benefit.id]}
				<li class="flex gap-5 border-t border-border py-8">
					<span
						class="grid size-12 shrink-0 place-items-center rounded-xl bg-secondary text-primary"
					>
						<Icon class="size-6" aria-hidden="true" />
					</span>
					<div>
						<h3 class="text-xl font-bold text-slate-900">{benefit.title}</h3>
						<p class="mt-2 {body} text-slate-700">{benefit.text}</p>
					</div>
				</li>
			{/each}
		</ul>

		<blockquote
			class="mt-12 max-w-[34ch] text-[clamp(1.6rem,2.6vw,2.25rem)] leading-tight font-bold tracking-[-0.02em] text-balance text-primary"
		>
			{closing}
		</blockquote>
	</div>
</section>

<!-- Standards -->
<section class="bg-secondary">
	<div class="{container} py-20 md:py-28">
		<div class="grid gap-12 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] lg:gap-16">
			<div>
				<h2 class="{h2} text-primary">Standards you can check against:</h2>
				<p class="mt-6 {lead} text-slate-700">
					Credimi runs the reference suites for you, side by side, and adds what they
					lack: real wallets on real devices.
				</p>
				<a
					href={conformanceChecksHref}
					class="mt-6 inline-flex items-center gap-2 font-medium text-primary underline underline-offset-4"
				>
					<BadgeCheck class="size-5" aria-hidden="true" />
					See all conformance checks ↗
				</a>
			</div>
			<StandardsCoverage />
		</div>
	</div>
</section>

<!-- CTA -->
<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative py-20 md:py-28">
		<div class="max-w-[48rem]">
			<h2 class={h2}>Run your first conformance check</h2>
			<p class="mt-6 {lead} text-primary-foreground/85">
				Whether you build an EUDI wallet, issue credentials or verify them, start with one
				suite and add it to CI when it passes.
			</p>
			<div class="mt-9 flex flex-wrap gap-3">
				<Button href="/my/tests/new" size="lg" variant="secondary">Start a new test</Button>
				<a
					href="mailto:credimi@forkbomb.eu"
					class="inline-flex h-10 items-center rounded-md border border-primary-foreground/40 px-6 text-sm font-medium transition-colors duration-150 hover:bg-primary-foreground/10"
				>
					Talk to us
				</a>
			</div>
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
					<strong class="flex-1 text-lg leading-snug font-semibold text-slate-900">
						{item.title}
					</strong>
					<span class="font-medium text-primary">{item.action}</span>
				</a>
			{/each}
		</div>
	</div>
</section>
