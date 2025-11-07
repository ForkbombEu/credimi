<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';

	import { setDashboardNavbar } from '../../+layout@.svelte';

	let { data } = $props();
	let { pipeline } = $derived(data);

	$effect(() => {
		setDashboardNavbar({ title: `${m.Pipelines()} | ${m.Settings()} | ${pipeline.name}` });
	});
</script>

<pre>{JSON.stringify(pipeline, null, 2)}</pre>
