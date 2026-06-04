<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowRight } from '@lucide/svelte';
	import { Scoreboard } from '$lib';

	import PublicPageHeader from '@/components/layout/public-page-header.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	//

	let { data } = $props();

	const scoreboard = new Scoreboard.Instance({
		pageSize: 20,
		initialData: () => data.scoreboardData
	});
</script>

<div class="grow bg-secondary pt-0 pb-20">
	<PublicPageHeader entity="scoreboard" description={m.scoreboard_page_description()}>
		{#snippet titleRight()}
			<Button
				variant="outline"
				size="sm"
				href="/scoreboard/interop"
				class={[
					'border-primary bg-transparent text-xs hover:bg-primary/10',
					'gap-1 [&_svg]:size-2',
					'h-auto! w-auto! px-2 py-2'
				]}
			>
				{m.scoreboard_view_interop_matrix()}
				<ArrowRight />
			</Button>
		{/snippet}
	</PublicPageHeader>

	<div class="px-4">
		<Scoreboard.Component {scoreboard} />
	</div>
</div>
