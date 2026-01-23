<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { Pencil, Plus } from '@lucide/svelte';

	import type { WalletsResponse } from '@/pocketbase/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import WalletForm from './wallet-form.svelte';

	type Props = {
		walletId?: string;
		initialData?: WalletsResponse;
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		customTrigger?: Snippet<[{ sheetTriggerAttributes: any }]>;
	};

	let { walletId, initialData, customTrigger }: Props = $props();
</script>

<Sheet>
	{#snippet trigger({ sheetTriggerAttributes })}
		{#if customTrigger}
			{@render customTrigger({ sheetTriggerAttributes })}
		{:else}
			<Button
				variant={walletId ? 'outline' : 'default'}
				class={walletId ? 'p-2' : ''}
				{...sheetTriggerAttributes}
			>
				{#if walletId}
					<Pencil />
				{:else}
					<Plus />
					{m.Add_new_wallet()}
				{/if}
			</Button>
		{/if}
	{/snippet}

	{#snippet content({ closeSheet })}
		<div class="space-y-6">
			<T tag="h3">{walletId ? m.Edit_wallet() : m.Add_new_wallet()}</T>
			<WalletForm
				{walletId}
				{initialData}
				onSuccess={() => {
					closeSheet();
					// Wallet will be automatically updated via CollectionManager subscription
				}}
			/>
		</div>
	{/snippet}
</Sheet>
