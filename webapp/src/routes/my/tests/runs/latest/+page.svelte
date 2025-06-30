<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	let { data } = $props();
</script>

<div>
	<h1>Latest Check Runs</h1>
	<pre>{JSON.stringify(data.latestCheckRuns, null, 2)}</pre>
</div>
