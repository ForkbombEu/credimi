<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { m } from '@/i18n';

	import * as Column from '../column';
	import BaseHeader from './headers/base-header.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.expand.pipeline,
		id: 'name',
		header: Column.header(BaseHeader, {
			header: m.Pipeline()
		})
	});
</script>

<script lang="ts">
	import { getPath } from '$lib/utils';

	import A from '@/components/ui-custom/a.svelte';

	import Na from './partials/na.svelte';

	//

	let { value }: Column.Props<typeof column> = $props();

	const href = $derived(value ? `/marketplace/pipelines/${getPath(value)}` : null);
</script>

<div class="leading-none wrap-break-word whitespace-normal">
	{#if href && value}
		<A {href} class="text-xs font-bold">{value.name}</A>
	{:else}
		<Na />
	{/if}
</div>
