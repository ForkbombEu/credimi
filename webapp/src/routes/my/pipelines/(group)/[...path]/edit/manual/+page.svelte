<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import PipelineFormManual from '$lib/pipeline-form/pipeline-form-manual.svelte';

	let { data } = $props();
	let { pipeline } = $derived(data);
</script>

<PipelineFormManual recordId={pipeline.id} initialData={pipeline} />
