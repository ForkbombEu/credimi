<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { localizeHref, m } from '@/i18n';

	import * as Column from '../column';

	//

	export const column = Column.define({
		fn: (row) => row.expand.pipeline,
		id: 'name',
		header: m.Pipeline()
	});
</script>

<script lang="ts">
	import { getPath } from '$lib/utils';

	import A from '@/components/ui-custom/a.svelte';

	//

	let { value }: Column.Props<typeof column> = $props();

	const href = $derived(localizeHref(`/marketplace/pipelines/${getPath(value)}`));
</script>

<A {href}>{value.name}</A>
