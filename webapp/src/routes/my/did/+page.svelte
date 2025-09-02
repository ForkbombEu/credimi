<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CodeDisplay from '$lib/layout/codeDisplay.svelte';

	import T from '@/components/ui-custom/t.svelte';

	let { data } = $props();
	const did = JSON.stringify(data.did, null, 2);
</script>

<div class="space-y-4 p-4">
	<T tag="h3">Your DID</T>
	<CodeDisplay content={did} language="json" />
</div>
