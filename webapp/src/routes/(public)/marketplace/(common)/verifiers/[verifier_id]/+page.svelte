<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	let { data } = $props();
</script>

<div>
	<pre>{JSON.stringify(data.verifier, null, 2)}</pre>
</div>
