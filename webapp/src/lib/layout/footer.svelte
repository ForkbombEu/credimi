<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { AppLogo } from '@/brand';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import LanguageSelect from '@/i18n/languageSelect.svelte';
	import { appVersion } from '@/utils/appVersion';

	const footer_data = [
		{
			label: m.Platform(),
			links: [
				{ label: m.Overview(), url: 'https://docs.credimi.io/' },
				{
					label: m.Conformance_automation(),
					url: 'https://docs.credimi.io/Manual/compliance-checks.html'
				},
				{ label: m.api_documentation(), url: 'https://docs.credimi.io/API/index.html' },
				{
					label: m.platform_architecture(),
					url: 'https://docs.credimi.io/Software_Architecture/1_start.html'
				},
				{
					label: m.deployment_options(),
					url: 'https://docs.credimi.io/Software_Architecture/7_dev_setup.html'
				}

				//			,{ label: m.compliance_standards(), url: '/platform/compliance' }
			]
		},
		/*	{
			label: m.Solutions(),
			links: [
				{ label: m.didital_identity_for_governments(), url: '/solutions/government' },
				{ label: m.financial_services(), url: '/solutions/finance' },
				{ label: m.workforce_verification(), url: '/solutions/workforce' },
				{ label: m.education_credentials(), url: '/solutions/education' },
				{ label: m.transportation_smart_mobility(), url: '/solutions/transportation' }
			]
		},
	*/
		{
			label: m.Capabilities(),
			links: [
				/*		
				{ label: m.credential_issuance(), url: '/capabilities/issuance' },
				{ label: m.credential_verification(), url: '/capabilities/verification' }, 
	*/
				{
					label: m.identity_wallet_integration(),
					url: 'https://docs.credimi.io/Manual/compliance-checks.html'
				},
				{
					label: m.conformance_compliance_tools(),
					url: 'https://docs.credimi.io/Software_Architecture/Workflow_YAML.html'
				},
				{
					label: m.ecosystem_governance(),
					url: 'https://docs.credimi.io/Manual/browse-marketplace.html'
				}
			]
		},
		{
			label: m.Developers(),
			links: [
				{ label: m.get_started_with_api(), url: 'https://docs.credimi.io/API/index.html' },
				//			{ label: m.sdks_libraries(), url: '/developers/sdks' },
				//			{ label: m.sandbox_environment(), url: '/developers/sandbox' },
				{
					label: m.test_suites_runs(),
					url: 'https://docs.credimi.io/Manual/compliance-checks.html'
				},
				{
					label: m.developer_tools(),
					url: 'https://docs.credimi.io/Software_Architecture/Workflow_YAML.html'
				}
				//			{ label: m.Changelog(), url: '/developers/changelog' }
			]
		},
		{
			label: m.Resources(),
			links: [
				{ label: m.Documentation(), url: 'https://docs.credimi.io/' },
				{ label: m.articles_tutorials(), url: 'https://docs.credimi.io/' },
				{ label: m.videos(), url: 'https://docs.credimi.io/' }
				//			{ label: m.case_studies(), url: '/resources/case-studies' },
				//			{ label: m.press_releases(), url: '/resources/press' },
				//			{ label: m.Blog(), url: '/blog' }
			]
		},
		{
			label: m.Community(),
			links: [
				{ label: m.join_our_slack(), url: '/community/slack' },
				{ label: 'GitHub', url: 'https://github.com/forkbombeu/credimi' },
				{ label: m.events_webinars(), url: '/community/events' },
				{ label: m.conformance_community_forum(), url: '/community/forum' },
				{ label: m.partner_network(), url: '/partners' }
			]
		},
		{
			label: m.Company(),
			links: [
				{ label: m.about_credimi(), url: 'https://forkbomb.solutions/about-us/' },
				//			{ label: m.our_vision_and_approach(), url: '/about/approach' },
				{ label: m.careers(), url: 'https://forkbomb.solutions/about-us/' },
				//			{ label: m.security(), url: '/security' },
				{ label: m.contact_us(), url: 'mailto:credimi@forkbomb.eu' },
				{
					label: m.legal_privacy(),
					url: 'https://docs.credimi.io/Legal/privacy-policy.html'
				},
				{
					label: m.terms_and_conditions(),
					url: 'https://docs.credimi.io/Legal/terms-and-conditions.html'
				}
			]
		}
	];
</script>

<div class="bg-primary text-muted">
	<div class="mx-auto max-w-screen-xl">
		<div class="px-8 py-12">
			<div class="flex flex-col justify-between gap-6 sm:flex-row">
				<div class="flex flex-col items-start gap-2">
					<div class="flex flex-row items-center gap-2">
						<AppLogo version="full" />
						<T tag="h4">
							<span class="pl-1 text-sm text-white/40">{appVersion}</span>
						</T>
					</div>
					<p class="border-b-muted border-b">{m.Test_and_find_decentralized_IDs()}</p>
				</div>
				<div class="flex flex-col gap-2 sm:flex-row">
					<LanguageSelect />
					<Button variant="ghost" class="grow basis-1 border sm:grow-0" href="/login">
						{m.Login()}
					</Button>
				</div>
			</div>
			<div class="grid grid-cols-1 gap-12 pt-12 sm:grid-cols-2 md:grid-cols-4">
				{#each footer_data as category (category.label)}
					<div class="flex flex-row gap-2">
						<div class="flex flex-col gap-2">
							<T tag="small">{category.label}</T>
							{#each category.links as item (item.label)}
								<!-- eslint-disable-next-line svelte/no-navigation-without-resolve, svelte/no-navigation-without-resolve -->
								<a href={item.url} class="text-muted-foreground text-xs">
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
