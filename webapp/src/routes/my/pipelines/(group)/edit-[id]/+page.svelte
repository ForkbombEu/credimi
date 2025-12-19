<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { PipelineForm } from '$pipeline-form/pipeline-form.svelte.js';

	//

	const { data } = $props();
	const form = new PipelineForm({ mode: 'edit', pipeline: data.pipeline });
</script>

<form.Component {form} />
