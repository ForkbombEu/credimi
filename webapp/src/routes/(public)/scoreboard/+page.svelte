<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ChartNoAxesCombined } from '@lucide/svelte';
	import { Scoreboard } from '$lib';

	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	//

	let { data } = $props();

	const scoreboard = new Scoreboard.Instance({
		pageSize: 20,
		initialData: () => data.scoreboardData
	});
</script>

<div class="grow bg-secondary pt-0 pb-20">
	<div
		class="mx-auto mb-8 flex max-w-7xl items-center justify-center gap-4 px-4 pt-10 text-primary md:px-8"
	>
		<ChartNoAxesCombined class="size-10" />
		<T tag="h1">
			{m.Scoreboard()}
		</T>
	</div>
	<div class="px-4">
		<Scoreboard.Component {scoreboard} />
	</div>
</div>
