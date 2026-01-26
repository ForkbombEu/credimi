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
	<h1 class="page-title">{m.public_scoreboard()}</h1>
	<p class="page-description">{m.public_scoreboard_description()}</p>

	<ScoreboardTableTabbed data={data.scoreboardData} showActions={false} />

	<OTelDetails content={data.scoreboardData.otelData} />
</div>

<style lang="postcss">
	@reference "tailwindcss";

	.page-container {
		@apply container mx-auto p-6;
	}

	.page-title {
		@apply mb-2 text-3xl font-bold;
	}

	.page-description {
		@apply mb-6 text-gray-600;
	}
</style>
