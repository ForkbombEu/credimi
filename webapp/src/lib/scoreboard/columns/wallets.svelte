<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { entities } from '$lib/global';

	import { renderComponent } from '@/components/ui/data-table';

	import * as Column from '../column';
	import * as EntityDisplay from '../entity-display';
	import EntityHeader from './headers/entity-header.svelte';

	export const column = Column.define({
		fn: (row) => {
			const wallets = row.expand.wallets ?? [];
			const walletVersions = row.expand.wallet_versions ?? [];

			return EntityDisplay.fromWalletRows(
				wallets.map((wallet) => ({
					wallet,
					version: walletVersions.find((version) => version.wallet === wallet.id)
				}))
			);
		},
		id: 'wallets',
		header: renderComponent(EntityHeader, {
			data: entities.wallets
		}),
		sortField: 'wallets.name',
		manualPillPositioning: true
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<EntityDisplay.List items={value} layout="full" />
