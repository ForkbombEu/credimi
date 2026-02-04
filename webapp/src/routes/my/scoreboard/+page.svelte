<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { OTelDetails, ScoreboardTableTabbed } from '$lib/scoreboard';

	import { m } from '@/i18n';

	import type { PageData } from './$types';

	//

	let { data }: { data: PageData } = $props();
</script>

<div class="page-container">
	<h1 class="page-title">{m.scoreboard()}</h1>

	<ScoreboardTableTabbed data={data.scoreboardData} showActions={true} />

	<OTelDetails content={data.scoreboardData.otelData} />
</div>

<style lang="postcss">
	@reference "tailwindcss";

	.page-container {
		@apply container mx-auto p-6;
	}

	.page-title {
		@apply mb-6 text-3xl font-bold;
	}
</style>
