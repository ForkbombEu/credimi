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
			return { wallets };
		},
		id: 'wallets',
		header: m.Wallet()
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

{JSON.stringify(value)}
