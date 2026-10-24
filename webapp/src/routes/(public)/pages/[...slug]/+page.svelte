<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { entities } from '$lib/global/entities';
	import RoleTile from '$lib/layout/marketing/role-tile.svelte';

	import Button from '@/components/ui-custom/button.svelte';

	const { data } = $props();
	const { attributes, body, headings } = $derived(data);
	const lead = $derived(attributes.lead ?? attributes.description);

	// Same frame as /pages/why-credimi: indigo hero, then full-width sections with alternating washes.
	const container = 'mx-auto max-w-screen-xl px-4 md:px-8';

	// Each H2 opens a section. A section of plain text puts its heading beside the text on wide
	// screens; diagrams, tables and code keep the full width. "Read next" closes the page.
	const sections = $derived(
		body.split(/(?=<h2[\s>])|(?=<div class="content-related">)/).map((html) => {
			const match = html.match(/^(<h2[\s\S]*?<\/h2>)([\s\S]*)$/);
			return {
				html,
				heading: match?.[1],
				text: match?.[2] ?? '',
				split: Boolean(match) && !/<(figure|table|pre)[\s>]/.test(html),
				related: html.startsWith('<div class="content-related">')
			};
		})
	);
	const mainSections = $derived(sections.filter((section) => !section.related));
	const related = $derived(sections.find((section) => section.related));
</script>

<!-- Hero -->
<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative pt-16 pb-12 md:pt-24 md:pb-16">
		<div
			class="grid grid-cols-[minmax(0,1fr)] items-center gap-14 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)] lg:gap-16"
		>
			<div>
				<a
					href="/pages/guides"
					class="text-xs font-medium tracking-[0.14em] text-primary-foreground/70 uppercase transition-colors duration-150 hover:text-primary-foreground"
				>
					Guides
				</a>
				<h1
					class="mt-4 text-[clamp(2.5rem,5.2vw,4.5rem)] leading-[1.04] font-bold tracking-[-0.03em] text-balance"
				>
					{attributes.title}
				</h1>
				{#if lead}
					<p
						class="mt-7 max-w-[58ch] text-xl leading-relaxed text-pretty text-primary-foreground/85"
					>
						{lead}
					</p>
				{/if}
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
				<p class="mt-8 text-sm text-primary-foreground/70">
					By the Credimi team at Forkbomb BV · Updated
					<span class="font-mono">{attributes.updatedOn.toLocaleDateString('en-GB')}</span
					>
				</p>
			</div>

			{#if attributes.heroCards}
				<ol class="hero-cards relative space-y-4">
					{#each attributes.heroCards as card, i (card.title)}
						<li
							class="relative flex items-center gap-4 rounded-2xl bg-card p-5 text-card-foreground"
						>
							<RoleTile role={entities[card.role]} />
							<div class="min-w-0">
								<p class="text-lg leading-tight font-bold">{card.title}</p>
								<p
									class={[
										'mt-1 font-mono text-sm',
										entities[card.role].classes.text
									]}
								>
									{card.detail}
								</p>
							</div>
							{#if i < attributes.heroCards.length - 1}
								<span class="hero-link" aria-hidden="true"></span>
							{/if}
						</li>
					{/each}
				</ol>
			{/if}
		</div>

		{#if attributes.facts}
			<dl
				class="mt-16 grid divide-primary-foreground/15 border-t border-primary-foreground/15 sm:grid-cols-2 lg:grid-cols-4 lg:divide-x"
			>
				{#each attributes.facts as fact (fact.term)}
					<div class="py-5 lg:px-6 lg:first:pl-0">
						<dt class="text-sm text-primary-foreground/70">{fact.term}</dt>
						<dd class="mt-1 font-semibold">{fact.value}</dd>
					</div>
				{/each}
			</dl>
		{/if}
	</div>
</section>

{#if headings.length}
	<nav aria-label="On this page" class="sticky top-0 z-30 border-b border-border bg-card">
		<div class="{container} flex items-center gap-2 overflow-x-auto py-3">
			<span class="mr-2 shrink-0 text-sm font-medium text-slate-600">On this page:</span>
			{#each headings as heading (heading.id)}
				<a
					href="#{heading.id}"
					class="shrink-0 rounded-full border border-border px-3 py-1 text-sm whitespace-nowrap text-slate-700 transition-colors duration-150 ease-out hover:border-primary hover:text-primary"
				>
					{heading.text}
				</a>
			{/each}
		</div>
	</nav>
{/if}

{#each mainSections as section, i (i)}
	<section class={i % 2 ? 'bg-secondary' : 'bg-background'}>
		<div class="{container} guide-body py-16 md:py-24" class:guide-intro={!section.heading}>
			{#if section.split}
				<div class="grid gap-x-16 lg:grid-cols-[minmax(0,5fr)_minmax(0,7fr)]">
					<div class="guide-aside">
						<!-- eslint-disable-next-line svelte/no-at-html-tags -->
						{@html section.heading}
					</div>
					<div>
						<!-- eslint-disable-next-line svelte/no-at-html-tags -->
						{@html section.text}
					</div>
				</div>
			{:else}
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				{@html section.html}
			{/if}
		</div>
	</section>
{/each}

{#if attributes.cta}
	<section class="fold-motif bg-primary text-primary-foreground">
		<div
			class="{container} relative grid items-center gap-10 py-20 md:py-24 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)]"
		>
			<div>
				<h2
					class="text-[clamp(2rem,3.6vw,3.25rem)] leading-[1.08] font-bold tracking-[-0.025em] text-balance"
				>
					{attributes.cta.title}
				</h2>
				<p class="mt-5 max-w-[52ch] text-xl leading-relaxed text-primary-foreground/85">
					{attributes.cta.text}
				</p>
			</div>
			<div class="flex flex-wrap gap-3 lg:justify-end">
				<Button href={attributes.cta.href} size="lg" variant="secondary">
					{attributes.cta.label}
				</Button>
				<a
					href="mailto:credimi@forkbomb.eu"
					class="inline-flex h-10 items-center rounded-md border border-primary-foreground/40 px-6 text-sm font-medium transition-colors duration-150 hover:bg-primary-foreground/10"
				>
					Talk to us
				</a>
			</div>
		</div>
	</section>
{/if}

{#if related}
	<section class="bg-secondary">
		<div class="{container} guide-body py-16">
			<!-- eslint-disable-next-line svelte/no-at-html-tags -->
			{@html related.html}
		</div>
	</section>
{/if}

<style>
	/* The signal travelling down the hero cards, as on /pages/why-credimi. */
	.hero-link {
		position: absolute;
		top: 100%;
		left: 2.75rem;
		width: 2px;
		height: 1rem;
		overflow: hidden;
		background: oklch(1 0 0 / 0.3);
	}

	.hero-link::after {
		position: absolute;
		inset: 0;
		background: var(--color-green-400);
		transform: translateY(-100%);
		animation: signal 1.8s cubic-bezier(0.16, 1, 0.3, 1) infinite;
		content: '';
	}

	.hero-cards li:nth-child(2) .hero-link::after {
		animation-delay: 0.6s;
	}

	@keyframes signal {
		to {
			transform: translateY(100%);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.hero-link::after {
			animation: none;
			transform: none;
		}
	}

	/* Markdown body: the type roles of /pages/why-credimi (display H2, lead, body). */
	.guide-body :global(h2) {
		max-width: 22ch;
		margin-bottom: 1.5rem;
		scroll-margin-top: 5rem;
		font-size: clamp(2rem, 3.6vw, 3.25rem);
		font-weight: 700;
		line-height: 1.08;
		letter-spacing: -0.025em;
		text-wrap: balance;
		color: var(--primary);
	}

	@media (min-width: 1024px) {
		.guide-aside {
			position: sticky;
			top: 5rem;
			align-self: start;
		}
	}

	.guide-body :global(h3) {
		max-width: 68ch;
		margin-top: 2.5rem;
		font-size: 1.375rem;
		font-weight: 700;
		line-height: 1.3;
		color: var(--color-slate-900);
	}

	.guide-body :global(:is(h3, .faq-item)) {
		scroll-margin-top: 5rem;
	}

	/* Shareable section links: shown on hover or focus, always on touch screens. */
	.guide-body :global(.heading-anchor) {
		margin-left: 0.35em;
		font-weight: 500;
		color: var(--muted-foreground);
		text-decoration: none;
		opacity: 0;
		transition: opacity 150ms ease-out;
	}

	.guide-body :global(:is(h2, h3, summary):hover .heading-anchor),
	.guide-body :global(.heading-anchor:focus-visible),
	.guide-body :global(:target .heading-anchor) {
		opacity: 1;
	}

	@media (hover: none) {
		.guide-body :global(.heading-anchor) {
			opacity: 0.6;
		}
	}

	.guide-body :global(:is(p, ul, ol)) {
		max-width: 68ch;
		margin-top: 1.25rem;
		font-size: 1.125rem;
		line-height: 1.65;
		text-wrap: pretty;
		color: var(--color-slate-700);
	}

	.guide-body :global(:is(p, ul, ol):first-child) {
		margin-top: 0;
	}

	.guide-intro > :global(p) {
		font-size: 1.25rem;
		line-height: 1.6;
	}

	.guide-body :global(ul) {
		padding-left: 1.5rem;
		list-style: disc;
	}

	.guide-body :global(ol) {
		padding-left: 1.5rem;
		list-style: decimal;
	}

	.guide-body :global(li + li) {
		margin-top: 0.5rem;
	}

	.guide-body :global(li::marker) {
		color: var(--primary);
	}

	.guide-body :global(strong) {
		font-weight: 600;
		color: var(--color-slate-900);
	}

	.guide-body :global(a:not([class])) {
		font-weight: 500;
		color: var(--primary);
		text-decoration: underline;
		text-underline-offset: 4px;
	}

	.guide-body :global(pre) {
		/* Code sections are full width; 68ch of body text is too narrow for 0.875rem code. */
		max-width: 52rem;
		margin-top: 1.25rem;
		padding: 1.25rem 1.5rem;
		overflow-x: auto;
		border: 1px solid var(--border);
		border-radius: 10px;
		background: var(--card) !important;
		font-size: 0.875rem;
		line-height: 1.6;
	}

	.guide-body :global(pre code) {
		padding: 0;
		background: none;
		color: var(--color-slate-800);
	}

	.guide-body :global(table) {
		width: 100%;
		max-width: 64rem;
		margin-top: 1.5rem;
		overflow: hidden;
		border-collapse: separate;
		border-spacing: 0;
		border: 1px solid var(--border);
		border-radius: 10px;
		background: var(--card);
		font-size: 0.95rem;
		line-height: 1.5;
	}

	.guide-body :global(:is(th, td)) {
		padding: 0.75rem 1rem;
		border-bottom: 1px solid var(--border);
		text-align: left;
		vertical-align: top;
		color: var(--color-slate-700);
	}

	.guide-body :global(th) {
		font-size: 0.75rem;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--color-slate-900);
		background: var(--muted);
	}

	.guide-body :global(tr:last-child td) {
		border-bottom: 0;
	}

	@media (max-width: 640px) {
		.guide-body :global(table) {
			display: block;
			overflow-x: auto;
		}
	}

	.guide-body :global(code) {
		padding: 0.1em 0.35em;
		border-radius: 4px;
		background: var(--secondary);
		font-family: 'Source Code Pro', ui-monospace, monospace;
		font-size: 0.9em;
		color: var(--primary);
	}

	.guide-body :global(figure) {
		margin: 3rem 0;
		overflow: hidden;
		border: 1px solid var(--border);
		border-radius: 1rem;
		background: var(--card);
	}

	.guide-body :global(figure img) {
		display: block;
		width: 100%;
		height: auto;
	}

	.guide-body :global(figcaption) {
		padding: 0.875rem 1.25rem;
		border-top: 1px solid var(--border);
		font-size: 0.875rem;
		color: var(--muted-foreground);
	}

	/* FAQ: each question is a <details> built by $lib/content. */
	.guide-body :global(.faq-item) {
		max-width: 68ch;
		border-bottom: 1px solid var(--border);
	}

	.guide-body :global(.faq-item summary) {
		display: flex;
		justify-content: space-between;
		gap: 1.5rem;
		padding: 1.25rem 0;
		font-size: 1.25rem;
		font-weight: 600;
		line-height: 1.35;
		color: var(--color-slate-900);
		cursor: pointer;
		list-style: none;
	}

	.guide-body :global(.faq-item summary::-webkit-details-marker) {
		display: none;
	}

	.guide-body :global(.faq-item summary::after) {
		flex-shrink: 0;
		color: var(--primary);
		content: '+';
	}

	.guide-body :global(.faq-item[open] summary::after) {
		content: '−';
	}

	.guide-body :global(.faq-item summary:focus-visible) {
		outline: 2px solid var(--primary);
		outline-offset: 4px;
	}

	.guide-body :global(.faq-item > p) {
		margin: 0 0 1.25rem;
	}

	.guide-body :global(.content-quote) {
		margin: 3rem 0;
		padding-left: 1.5rem;
		border-left: 2px solid var(--primary);
	}

	.guide-body :global(.content-quote p) {
		max-width: 44ch;
		margin: 0;
		font-size: clamp(1.5rem, 2.4vw, 2rem);
		font-weight: 700;
		line-height: 1.25;
		letter-spacing: -0.02em;
		text-wrap: balance;
		color: var(--primary);
	}

	.guide-body :global(.content-related-title) {
		max-width: none;
		margin: 0;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid var(--border);
		font-size: 1.5rem;
		font-weight: 700;
		color: var(--primary);
	}

	.guide-body :global(.content-related-grid) {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(16rem, 1fr));
		gap: 1rem;
		margin-top: 1.5rem;
	}

	.guide-body :global(.content-related-grid a) {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 1.5rem;
		border: 1px solid var(--border);
		border-radius: 1rem;
		background: var(--card);
		text-decoration: none;
		transition: border-color 150ms ease-out;
	}

	.guide-body :global(.content-related-grid a:hover) {
		border-color: var(--primary);
	}

	.guide-body :global(.content-related-grid span) {
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--color-slate-600);
	}

	.guide-body :global(.content-related-grid strong) {
		flex: 1;
		font-size: 1.125rem;
		line-height: 1.35;
	}

	.guide-body :global(.content-related-grid em) {
		font-style: normal;
		font-weight: 500;
		color: var(--primary);
	}
</style>
