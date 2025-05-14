<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script>
	import CustomCheckForm from '../_partials/custom-check-form.svelte';

	let { data } = $props();
</script>

<CustomCheckForm standardsAndTestSuites={data.standardsAndTestSuites} />
