<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
import { ScoreboardTableTabbed } from '$lib/scoreboard';
import type { PageData } from './$types';

let { data }: { data: PageData } = $props();
</script>

<div class="container mx-auto p-6">
<h1 class="mb-6 text-3xl font-bold">Scoreboard</h1>

<ScoreboardTableTabbed data={data.scoreboardData} showActions={true} />

<!-- OpenTelemetry Data Section -->
<div class="mt-8">
<details class="rounded-lg border">
<summary class="cursor-pointer bg-gray-50 px-6 py-3 font-semibold">
OpenTelemetry Data (Click to expand)
</summary>
<div class="p-6">
<pre class="overflow-x-auto rounded bg-gray-100 p-4 text-sm">{JSON.stringify(
data.scoreboardData.otelData,
null,
2
)}</pre>
</div>
</details>
</div>
</div>
