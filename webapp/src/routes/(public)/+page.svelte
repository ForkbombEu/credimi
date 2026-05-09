<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Sparkle, StoreIcon, TableIcon } from '@lucide/svelte';
	import PageContent from '$lib/layout/pageContent.svelte';
	import PageTop from '$lib/layout/pageTop.svelte';

	import type { MarketplaceItemsResponse } from '@/pocketbase/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Badge } from '@/components/ui/badge';
	import { featureFlags } from '@/features';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import { MarketplaceItemCard } from '../../lib/marketplace';
	import ScoreboardSection from './_partials/scoreboard-section.svelte';

	//

	let { data } = $props();
	const { wallets, issuers, verifiers, scoreboard } = $derived(data);
</script>

{#if $featureFlags.DEMO}
	<div class="flex justify-end px-6 pt-4"></div>
{/if}

<PageTop>
	<div class="space-y-2">
		<T tag="h1" class="text-balance">
			{m.EUDIW_Conformance_Interoperability_and_Marketplace()}
		</T>
		<div class="flex flex-col gap-2 py-2">
			<T tag="h3" class="font-normal! text-balance">
				{m.Explore_the_marketplace_and_try_credentials_wallets_and_services()}
			</T>
			<T tag="h3" class="font-normal! text-balance">
				{m.Test_the_conformance_and_interoperability_of_your_EUDIW()}
			</T>
		</div>
	</div>
	<div class="flex flex-col gap-4 md:flex-row">
		<Button variant="default" href="/marketplace">
			{m.Explore_Marketplace()}
		</Button>
		<Button variant="secondary" href={$currentUser ? '/my/pipelines' : '/login'}>
			<Icon src={Sparkle} />
			{m.automated_conformance_interop()}
			<Badge variant="outline" class="border-primary text-xs text-primary">
				{m.Beta()}
			</Badge>
		</Button>
	</div>
</PageTop>

<PageContent class="bg-secondary" contentClass="space-y-12">
	<div class="space-y-6">
		<div class="flex items-center justify-between">
			<T tag="h3">{m.Find_solutions()}</T>
			<Button variant="default" href="/marketplace">
				<StoreIcon />
				{m.Explore_Marketplace()}
			</Button>
		</div>

		<div class="space-y-4">
			{@render row(issuers, 0)}
			{@render row(verifiers, 1)}
			{@render row(wallets, 2)}
		</div>
	</div>

	<div class="space-y-6">
		<div class="flex items-center justify-between">
			<T tag="h3">{m.Compare_by_test_results()}</T>
			<Button variant="default" href="/scoreboard">
				<TableIcon />
				{m.View_Scoreboard()}
			</Button>
		</div>
		<ScoreboardSection {...scoreboard} />
	</div>
</PageContent>

{#snippet row(items: MarketplaceItemsResponse[], index: number)}
	<!-- Try: https://stackoverflow.com/questions/22955465/overflow-y-scroll-is-hiding-overflowing-elements-on-the-horizontal-line -->
	<div
		class={[
			'-mx-2 -mt-3 scrollbar-none overflow-x-scroll px-2 pt-3',
			{
				'md:-translate-x-3': index === 0 || index === 2,
				'md:translate-x-3': index === 1
			}
		]}
	>
		<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
			{#each items as item (item.id)}
				<MarketplaceItemCard {item} />
			{/each}
		</div>
	</div>
{/snippet}

<!-- 
<PageContent class="border-y-primaryborder-y-2" contentClass="!space-y-8">
	<div id="waitlist" class="scroll-mt-20">
		 <T tag="h2" class="text-balance"> 
		 	{m._Stay_Ahead_in_Digital_Identity_Compliance_Join_Our_Early_Access_List()} 
		 </T> 
		 <T class="mt-1 text-balance font-medium"> 
		 	{m.Be_the_first_to_explore_credimi_the_ultimate_compliance_testing_tool_for_decentralized_identity_Get_exclusive_updates_early_access_and_a_direct_line_to_our_team_()} 
		 </T> 
	</div>

	 {#if !formSuccess} 
	 	<Form {form} hide={['submit_button']} class=" !space-y-3" hideRequiredIndicator> 
	 		<div class="flex w-full max-w-3xl flex-col gap-2 md:flex-row md:gap-6"> 
	 			<div class="grow"> 
	 				<Field 
	 					{form} 
	 					name="name" 
	 					options={{ 
	 						label: m.Your_name(), 
	 						placeholder: m.John_Doe(), 
	 						class: 'bg-secondary/40 ' 
	 					}} 
	 				/> 
	 			</div> 
	 			<div class="grow"> 
	 				<Field 
	 					{form} 
	 					name="email" 
	 					options={{ 
	 						label: m.Your_email(), 
	 						placeholder: m.e_g_hellomycompany_com(), 
	 						class: 'bg-secondary/40' 
	 					}} 
	 				/> 
	 			</div> 
	 		</div> 
	 		<SubmitButton>{m.Join_the_Waitlist()}</SubmitButton> 
	 	</Form> 
	 {:else} 
	 	<Alert variant="info"> 
	 		<p class="font-bold">{m.Request_sent_()}</p> 
	 		<p> 
	 			{m.Thanks_for_your_interest_We_will_write_to_you_soon()} 
	 		</p> 
	 	</Alert> 
	 {/if} 
</PageContent>
-->
