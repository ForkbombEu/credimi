<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => {
			const wallets = row.expand.wallets ?? [];
			const walletVersions = row.expand.wallet_versions ?? [];

			return wallets.map((wallet) => ({
				wallet,
				version: walletVersions.find((version) => version.wallet === wallet.id)
			}));
		},
		id: 'wallets',
		header: m.Wallet()
	});
</script>

<script lang="ts">
	import { getPath } from '$lib/utils';

	import A from '@/components/ui-custom/a.svelte';

	import Avatar from './partials/avatar.svelte';

	let { value }: Column.Props<typeof column> = $props();
</script>

<div class="flex flex-col gap-1">
	{#each value as item (item.wallet.id)}
		<div class="flex items-center gap-2">
			<Avatar record={item.wallet} />
			<div>
				<A href={`/marketplace/wallets/${getPath(item.wallet)}`}>
					{item.wallet.name}
				</A>
				{#if item.version}
					<p class="text-xs text-muted-foreground">
						{item.version.tag}
					</p>
				{/if}
			</div>
		</div>
	{/each}
</div>
