<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import SquareArrowOutUpRight from '@lucide/svelte/icons/square-arrow-out-up-right';
	import { layers, statusEntryExample, tools, workflow, type ExtraId } from '$lib/content/extras';
	import { extras } from '$lib/layout/topbar-links';

	import Button from '@/components/ui-custom/button.svelte';

	const { data } = $props();

	// Same type roles as the other marketing pages: display, section heading, lead, body.
	const container = 'mx-auto max-w-site px-4 md:px-8';
	const h2 =
		'text-[clamp(2rem,3.6vw,3.25rem)] leading-[1.08] font-bold tracking-[-0.025em] text-balance';
	const lead = 'text-xl leading-relaxed text-pretty';
	const body = 'text-lg leading-[1.65] text-pretty';

	// One color per tool, with its icon, so a tool is recognisable wherever the page names it.
	const styles: Record<ExtraId, { bg: string; text: string }> = {
		'capture-wallet': { bg: 'bg-blue-100', text: 'text-blue-800' },
		'capture-issuer-verifier': { bg: 'bg-green-100', text: 'text-green-800' },
		'trust-inspector': { bg: 'bg-amber-100', text: 'text-amber-800' },
		lote: { bg: 'bg-orange-100', text: 'text-orange-800' },
		tsl: { bg: 'bg-red-100', text: 'text-red-800' },
		federation: { bg: 'bg-purple-100', text: 'text-purple-800' },
		attestbook: { bg: 'bg-teal-100', text: 'text-teal-800' },
		atlas: { bg: 'bg-secondary', text: 'text-primary' }
	};

	const byId = Object.fromEntries(extras.map((extra) => [extra.id, extra])) as Record<
		ExtraId,
		(typeof extras)[number]
	>;

	const external = { target: '_blank', rel: 'noopener noreferrer' };
</script>

<!-- Hero -->
<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative pt-16 pb-16 md:pt-24 md:pb-24">
		<div class="space-y-14">
			<div>
				<h1
					class="text-[clamp(2.5rem,5.2vw,4.5rem)] leading-[1.04] font-bold tracking-[-0.03em] text-balance"
				>
					See what your identity stack is doing.
				</h1>
				<p class="mt-7 max-w-[60ch] {lead} text-primary-foreground/85">
					Credimi Extras are eight free, open-source tools that cover an EUDI Wallet
					exchange end to end: the messages, the trust behind them, and the rules they
					follow.
				</p>
				<div class="mt-9 flex flex-wrap gap-3">
					<Button href="#inspect" size="lg" variant="secondary"
						>Meet the eight tools</Button
					>
					<Button
						href="#workflow"
						size="lg"
						variant="outline"
						class="border-primary-foreground/40 bg-transparent text-primary-foreground hover:bg-primary-foreground/10 hover:text-primary-foreground"
					>
						Follow a debugging run
					</Button>
				</div>
			</div>

			<!-- Shortcuts: every tool, in the layer it looks at, one click from its hosted app. -->
			<nav aria-label="Open an Extra" class="grid gap-3 lg:grid-cols-3">
				{#each layers as layer (layer.id)}
					<div class="rounded-2xl bg-card p-5 text-card-foreground">
						<p class="font-semibold text-slate-900">{layer.title}</p>
						<ul class="mt-3 space-y-2">
							{#each layer.tools as id (id)}
								{@const style = styles[id]}
								{@const Icon = byId[id].icon}
								<li>
									<a
										href={byId[id].href}
										{...external}
										class="flex h-full items-start gap-3 rounded-xl p-3 transition-[filter] duration-150 ease-out hover:brightness-95 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary {style.bg}"
									>
										<span
											class="grid size-9 shrink-0 place-items-center rounded-lg bg-card {style.text}"
										>
											<Icon class="size-5" aria-hidden="true" />
										</span>
										<span class="min-w-0 flex-1">
											<span
												class="block leading-tight font-semibold {style.text}"
											>
												{byId[id].title}
											</span>
											<span
												class="mt-1 block text-sm leading-snug text-slate-700"
											>
												{tools[id].problem}
											</span>
										</span>
										<SquareArrowOutUpRight
											class="mt-0.5 size-4 shrink-0 {style.text}"
											aria-hidden="true"
										/>
									</a>
								</li>
							{/each}
						</ul>
					</div>
				{/each}
			</nav>
		</div>
	</div>
</section>

<!-- The eight tools, one row each -->
{#each layers as layer, l (layer.id)}
	<section id={layer.id} class={['scroll-mt-20', l % 2 ? 'bg-secondary' : 'bg-background']}>
		<div class="{container} py-20 md:py-28">
			<h2 class="{h2} max-w-[20ch] text-primary">{layer.title}</h2>
			<p class="mt-6 max-w-[60ch] {lead} text-slate-700">{layer.text}</p>

			{#each layer.tools as id, i (id)}
				{@const tool = tools[id]}
				{@const style = styles[id]}
				{@const Icon = byId[id].icon}
				{@const { href, repo, title } = byId[id]}
				<article
					{id}
					class="mt-16 grid scroll-mt-24 grid-cols-[minmax(0,1fr)] items-center gap-10 lg:grid-cols-2 lg:gap-16"
				>
					<div class={[i % 2 && 'lg:order-last']}>
						<h3
							class="flex items-center gap-4 text-2xl leading-tight font-bold tracking-[-0.02em] sm:text-3xl {style.text}"
						>
							<span
								class="grid size-14 shrink-0 place-items-center rounded-2xl {style.bg}"
							>
								<Icon class="size-7" aria-hidden="true" />
							</span>
							<span class="min-w-0 wrap-anywhere">{title}</span>
						</h3>
						<p
							class="mt-6 text-2xl leading-snug font-semibold text-balance text-slate-900"
						>
							{tool.problem}
						</p>
						<p class="mt-4 max-w-[60ch] {body} text-slate-700">{tool.does}</p>

						<dl
							class="mt-8 grid max-w-[60ch] grid-cols-[4rem_minmax(0,1fr)] gap-y-3 border-t border-border pt-6 leading-snug"
						>
							<dt class="font-semibold {style.text}">In</dt>
							<dd class="text-slate-800">{tool.input}</dd>
							<dt class="font-semibold {style.text}">Out</dt>
							<dd class="text-slate-800">{tool.output}</dd>
							<dt class="font-semibold {style.text}">For</dt>
							<dd class="text-slate-800">{tool.for}</dd>
						</dl>

						<div class="mt-8 flex items-center gap-5">
							<Button {href} {...external} size="lg" aria-label="Open {title}">
								Open tool
								<SquareArrowOutUpRight aria-hidden="true" />
							</Button>
							<a
								href={repo}
								{...external}
								aria-label="{title} on GitHub"
								class="font-medium text-primary underline underline-offset-4"
							>
								GitHub
							</a>
						</div>
						<p class="mt-5 max-w-[60ch] text-sm text-slate-600">{tool.limit}</p>
					</div>

					<figure class="rounded-3xl p-3 sm:p-5 {style.bg}">
						<img
							src="/pages/extras/{id}.webp"
							alt={tool.screen}
							width="1200"
							height="750"
							loading="lazy"
							class="w-full rounded-xl border border-border"
						/>
					</figure>
				</article>
			{/each}
		</div>
	</section>
{/each}

<!-- One debugging run -->
<section id="workflow" class="scroll-mt-20 bg-secondary">
	<div class="{container} py-20 md:py-28">
		<h2 class="{h2} max-w-[22ch] text-primary">
			A verifier keeps accepting a revoked credential
		</h2>
		<p class="mt-6 max-w-[60ch] {lead} text-slate-700">
			One failure, three tools. Change only the credential's status.
		</p>

		<div
			class="mt-12 grid grid-cols-[minmax(0,1fr)] gap-10 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] lg:gap-16"
		>
			<ol class="space-y-4">
				{#each workflow as step, i (step.title)}
					<li class="flex gap-4 rounded-2xl bg-card p-5">
						<span
							class="grid size-9 shrink-0 place-items-center rounded-full bg-primary font-mono text-sm font-semibold text-primary-foreground"
						>
							{i + 1}
						</span>
						<div>
							<h3 class="text-xl font-bold text-slate-900">{step.title}</h3>
							{#if step.tool === 'pipeline'}
								<a
									href="/pages/conformance-automation"
									class="mt-1 inline-block text-sm font-semibold text-primary underline underline-offset-4"
								>
									Credimi pipeline
								</a>
							{:else if step.tool === 'verifier'}
								<p class="mt-1 text-sm font-semibold text-slate-600">
									Your own verifier
								</p>
							{:else}
								<a
									href="#{step.tool}"
									class="mt-1 inline-block text-sm font-semibold underline underline-offset-4 {styles[
										step.tool
									].text}"
								>
									{byId[step.tool].title}
								</a>
							{/if}
							<p class="mt-2 leading-relaxed text-pretty text-slate-700">
								{step.text}
							</p>
						</div>
					</li>
				{/each}
			</ol>

			<div class="space-y-5">
				<figure class="overflow-hidden rounded-2xl border border-border bg-card">
					<pre
						class="overflow-x-auto px-5 py-4 font-mono text-sm leading-relaxed text-slate-800">{statusEntryExample}</pre>
					<figcaption class="border-t border-border px-5 py-3 text-sm text-slate-600">
						Example from the project README: the status entry a new test credential
						points to.
					</figcaption>
				</figure>
			</div>
		</div>
	</div>
</section>

<!-- FAQ -->
<section class="bg-background">
	<div class="{container} py-20 md:py-28">
		<div class="grid gap-10 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)] lg:gap-16">
			<h2 class="{h2} text-primary">What the Extras are not</h2>
			<div>
				{#each data.faq as item (item.question)}
					<details class="group border-b border-border">
						<summary
							class="flex cursor-pointer list-none justify-between gap-6 py-5 text-xl font-semibold text-slate-900 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-primary [&::-webkit-details-marker]:hidden"
						>
							{item.question}
							<span class="shrink-0 text-primary group-open:hidden" aria-hidden="true"
								>+</span
							>
							<span
								class="hidden shrink-0 text-primary group-open:inline"
								aria-hidden="true">−</span
							>
						</summary>
						<p class="max-w-[68ch] pb-5 text-lg leading-[1.65] text-slate-700">
							{item.answer}
						</p>
					</details>
				{/each}
			</div>
		</div>
	</div>
</section>

<!-- CTA -->
<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative py-20 md:py-28">
		<div class="max-w-[48rem]">
			<h2 class={h2}>Turn a finding into a regression test</h2>
			<p class="mt-6 {lead} text-primary-foreground/85">
				A Credimi pipeline reruns the scenario and keeps the evidence.
			</p>
			<div class="mt-9 flex flex-wrap gap-3">
				<Button href="/pages/conformance-automation" size="lg" variant="secondary">
					See how pipelines work
				</Button>
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
