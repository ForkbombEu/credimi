<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { CredentialsResponse, MarketplaceItemsResponse } from '@/pocketbase/types';

	import * as Accordion from '@/components/ui/accordion';
	import * as Table from '@/components/ui/table';
	import { localizeHref, m } from '@/i18n';

	import {
		getIssuerItemCredentials,
		getMarketplaceItemTypeData,
		isCredentialIssuer
	} from './utils';

	//

	type Props = {
		issuer: MarketplaceItemsResponse;
		show: boolean;
	};

	let { issuer, show }: Props = $props();

	let credentials = $state<CredentialsResponse[]>([]);

	$effect(() => {
		getIssuerItemCredentials(issuer).then((res) => {
			credentials = res;
		});
	});

	const {
		display: { icon: Icon }
	} = getMarketplaceItemTypeData('credentials');

	let accordionValue = $state<string | undefined>();
	const isOpen = $derived(Boolean(accordionValue));
</script>

<Table.Row class={['bg-gray-50', { hidden: !show, 'hide-previous-border': show }]}>
	{#if isCredentialIssuer(issuer)}
		<Table.Cell class="!p-0 py-1 text-xs" colspan={99}>
			<Accordion.Root type="single" bind:value={accordionValue}>
				<Accordion.Item value="item-1" class="border-none">
					<Accordion.Trigger
						class="flex justify-start gap-3 border-none px-4 py-1 hover:bg-slate-200 hover:no-underline"
					>
						<div class="flex w-[38px] justify-center">
							<Icon class="text-gray-400" size={12} />
						</div>
						<p class="py-1">
							{m.Credentials()}
							{#if credentials.length > 0}
								<span class="w-6">
									({credentials.length})
								</span>
							{/if}
						</p>
						<div class="flex w-0 grow gap-1 overflow-hidden">
							{#if credentials.length > 0 && !isOpen}
								{@const podium = credentials.slice(0, 3)}
								{@const rest = credentials.slice(3).length}

								{#each podium as credential}
									<a
										href={localizeHref(
											`/marketplace/credentials/${credential.id}`
										)}
										class="pill"
									>
										{credential.display_name}
									</a>
								{/each}
								{#if rest > 0}
									<span class="pill">
										+{rest}
									</span>
								{/if}
							{/if}
						</div>
					</Accordion.Trigger>

					{#if credentials.length > 0}
						<Accordion.Content>
							<div class="grid grid-cols-2 pl-[58px] md:grid-cols-3 lg:grid-cols-4">
								{#each credentials as credential}
									<a
										href={localizeHref(
											`/marketplace/credentials/${credential.id}`
										)}
										class="pill truncate text-xs"
									>
										{credential.display_name}
									</a>
								{/each}
							</div>
						</Accordion.Content>
					{/if}
				</Accordion.Item>
			</Accordion.Root>
		</Table.Cell>
	{/if}
</Table.Row>

<style lang="postcss">
	.pill {
		@apply block text-nowrap rounded-full px-2 py-1 text-slate-500 transition hover:bg-slate-300;
	}
</style>
