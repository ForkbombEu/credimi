<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { version } from '$app/environment';

	import { AppLogo } from '@/brand';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import LanguageSelect from '@/i18n/languageSelect.svelte';
	import { currentUser } from '@/pocketbase';

	import { extras } from './topbar-links';

	// One link per destination: product, guides, companion tools, company.
	const footer_data = [
		{
			label: m.Platform(),
			links: [
				{ label: m.Hub(), url: '/hub' },
				{ label: m.Scoreboard(), url: '/scoreboard' },
				{ label: 'Why Credimi', url: '/pages/why-credimi' },
				{ label: m.Documentation(), url: 'https://docs.credimi.io/' },
				{ label: m.api_documentation(), url: 'https://docs.credimi.io/API/index.html' }
			]
		},
		{
			label: 'Guides',
			links: [
				{ label: 'EUDIW conformance', url: '/pages/eudiw-conformance' },
				{ label: 'FCAF conformance', url: '/pages/fcaf-conformance' },
				{ label: 'EUDI wallet testing', url: '/pages/eudi-wallet-testing' },
				{
					label: 'EUDI interoperability testing',
					url: '/pages/eudi-wallet-interoperability-testing'
				},
				{ label: 'OpenID4VCI and OpenID4VP', url: '/pages/openid4vc-conformance' },
				{ label: 'All guides ↗', url: '/pages/guides' }
			]
		},
		{
			label: m.Extras(),
			links: extras.map(({ title, href }) => ({ label: title, url: href }))
		},
		{
			label: m.Company(),
			links: [
				{ label: m.about_credimi(), url: 'https://forkbomb.solutions/about-us/' },
				{ label: m.contact_us(), url: 'mailto:credimi@forkbomb.eu' },
				{ label: 'GitHub', url: 'https://github.com/forkbombeu/credimi' },
				{ label: m.legal_privacy(), url: 'https://docs.credimi.io/legal/privacy-policy' },
				{
					label: m.terms_and_conditions(),
					url: 'https://docs.credimi.io/legal/terms-and-conditions'
				}
			]
		}
	];
</script>

<div class="bg-primary text-muted">
	<div class="mx-auto max-w-site">
		<div class="px-8 py-12">
			<div class="flex flex-col justify-between gap-6 sm:flex-row">
				<div class="flex flex-col items-start gap-2">
					<div class="flex flex-row items-center gap-2">
						<AppLogo version="full" />
						<span class="pl-1 font-mono text-sm text-primary-foreground/70"
							>{version}</span
						>
					</div>
					<p class="max-w-md text-primary-foreground/85">{m.footer_tagline()}</p>
				</div>
				<div class="flex flex-col gap-2 sm:flex-row">
					<LanguageSelect />
					{#if !$currentUser}
						<Button variant="ghost" class="grow basis-1 border sm:grow-0" href="/login">
							{m.Login()}
						</Button>
					{/if}
				</div>
			</div>
			<div class="grid grid-cols-1 gap-12 pt-12 sm:grid-cols-2 md:grid-cols-4">
				{#each footer_data as category (category.label)}
					<div class="flex flex-row gap-2">
						<div class="flex flex-col gap-2">
							<T tag="small">{category.label}</T>
							{#each category.links as item (item.label)}
								<a
									href={item.url}
									class="text-sm text-primary-foreground/75 transition-colors duration-150 hover:text-primary-foreground"
								>
									{item.label}
								</a>
							{/each}
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
	<div class="flex items-center justify-center bg-black/30 p-2 px-8">
		<small>
			<a href="https://forkbomb.solutions/" class="underline hover:no-underline">
				{m.Proudly_developed_by_ForkBomb_BV()}
			</a>
		</small>
	</div>
</div>
