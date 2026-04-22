<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => row.expand.custom_integrations ?? [],
		id: 'custom_integrations',
		header: 'Custom integrations'
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

{JSON.stringify(value)}
