<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Scoreboard } from '$lib';

	import PublicPageHeader from '@/components/layout/public-page-header.svelte';
	import { m } from '@/i18n';

	//

	let { data } = $props();

	const scoreboard = new Scoreboard.Instance({
		pageSize: 20,
		initialData: () => data.scoreboardData
	});
</script>

<div class="grow bg-secondary pt-0 pb-20">
	<PublicPageHeader entity="scoreboard" description={m.scoreboard_page_description()} />
	<div class="px-4">
		<Scoreboard.Component {scoreboard} />
	</div>
</div>
