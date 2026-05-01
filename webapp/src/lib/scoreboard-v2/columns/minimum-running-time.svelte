<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => row.minimum_running_time,
		id: 'minimum_running_time',
		header: m.Min_running_time()
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

<span class="text-xs">
	{value}
</span>
