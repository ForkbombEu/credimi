<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	const { data } = $props();

	const container = 'mx-auto max-w-screen-xl px-4 md:px-8';
</script>

<section class="fold-motif bg-primary text-primary-foreground">
	<div class="{container} relative pt-16 pb-14 md:pt-24 md:pb-20">
		<p class="text-xs font-medium tracking-[0.14em] text-primary-foreground/70 uppercase">
			Guides
		</p>
		<h1
			class="mt-4 max-w-[20ch] text-[clamp(2.5rem,5.2vw,4.5rem)] leading-[1.04] font-bold tracking-[-0.03em] text-balance"
		>
			Test EUDI wallets, issuers and verifiers together
		</h1>
		<p class="mt-7 max-w-[58ch] text-xl leading-relaxed text-pretty text-primary-foreground/85">
			How conformance and interoperability testing works for the EUDI Wallet: FCAF, OpenID4VCI
			and OpenID4VP, eIDAS 2, and how Credimi runs them on real devices with evidence.
		</p>
	</div>
</section>

<section class="bg-secondary">
	<div class="{container} py-16 md:py-24">
		<ul class="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
			{#each data.guides as guide (guide.href)}
				<li>
					<a
						href={guide.href}
						class="flex h-full flex-col overflow-hidden rounded-2xl border border-border bg-card transition-colors duration-150 ease-out hover:border-primary"
					>
						{#if guide.image}
							<img
								src={guide.image}
								alt=""
								width="1200"
								height="630"
								loading="lazy"
								class="block aspect-[1200/630] w-full border-b border-border object-cover"
							/>
						{/if}
						<div class="flex flex-1 flex-col gap-3 p-6">
							<h2 class="text-xl leading-snug font-bold text-balance text-slate-900">
								{guide.title}
							</h2>
							{#if guide.lead}
								<p class="flex-1 leading-relaxed text-pretty text-slate-700">
									{guide.lead}
								</p>
							{/if}
							<span class="font-medium text-primary">Read guide ↗</span>
						</div>
					</a>
				</li>
			{/each}
		</ul>
	</div>
</section>
