<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IconComponent, Link } from '@/components/types';

	import Icon from '@/components/ui-custom/icon.svelte';
	import * as Accordion from '@/components/ui/accordion';
	import * as Table from '@/components/ui/table';
	import { localizeHref } from '@/i18n';

	//

	type Props = {
		items: Link[];
		title: string;
		icon: IconComponent;
		show: boolean;
	};

	let { items, title, icon, show }: Props = $props();

	let accordionValue = $state<string | undefined>();
	const isOpen = $derived(Boolean(accordionValue));
</script>

<Table.Row class={['bg-gray-50', { hidden: !show, 'hide-previous-border': show }]}>
	{#if show}
		<Table.Cell class="!p-0 py-1 text-xs" colspan={99}>
			<Accordion.Root type="single" bind:value={accordionValue}>
				<Accordion.Item value="item-1" class="border-none">
					<Accordion.Trigger
						class="flex justify-start gap-3 border-none px-4 py-1 hover:bg-slate-200 hover:no-underline"
					>
						<div class="flex w-[38px] justify-center">
							<Icon src={icon} class="text-gray-400" size={12} />
						</div>
						<p class="py-1">
							{title}
							{#if items.length > 0}
								<span class="w-6">
									({items.length})
								</span>
							{/if}
						</p>
						<div class="flex w-0 grow gap-1 overflow-hidden">
							{#if items.length > 0 && !isOpen}
								{@const podium = items.slice(0, 3)}
								{@const rest = items.slice(3).length}

								{#each podium as link (link.href)}
									<a href={localizeHref(link.href ?? '')} class="pill">
										{link.title}
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

					{#if items.length > 0}
						<Accordion.Content class="[&>div]:pb-1">
							<div class="grid grid-cols-2 pl-[58px] md:grid-cols-3 lg:grid-cols-4">
								{#each items as link (link.href)}
									<a
										href={localizeHref(link.href ?? '')}
										class="pill truncate text-xs"
									>
										{link.title}
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
