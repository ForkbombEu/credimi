<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => row.expand.use_case_verifications ?? [],
		id: 'use_case_verifications',
		header: 'Use case verifications'
	});
</script>

<script lang="ts">
	let { value }: Column.Props<typeof column> = $props();
</script>

{JSON.stringify(value)}
