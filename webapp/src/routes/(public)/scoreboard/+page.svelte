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
		class="mx-auto mb-8 flex max-w-7xl flex-col items-center justify-center gap-4 px-4 pt-10 text-primary md:px-8"
	>
		<div class="flex items-center gap-2">
			<ChartNoAxesCombined class="size-10" />
			<T tag="h1">
				{m.Scoreboard()}
			</T>
		</div>
		<p class="max-w-prose text-center text-balance">{m.scoreboard_page_description()}</p>
	</div>
	<div class="px-4">
		<Scoreboard.Component {scoreboard} />
	</div>
</div>
