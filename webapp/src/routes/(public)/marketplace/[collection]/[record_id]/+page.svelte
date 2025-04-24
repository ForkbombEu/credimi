<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import BackButton from '$lib/layout/back-button.svelte';
	import { m } from '@/i18n';

	let { data } = $props();
</script>

<BackButton href="/marketplace">
	{m.Back_to_marketplace()}
</BackButton>

<pre>{JSON.stringify(data, null, 2)}</pre>
