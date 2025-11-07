<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { PipelineForm } from '$pipeline-form/pipeline-form.svelte.js';

	//

	const { data } = $props();
	console.log('data', data);
	const form = new PipelineForm(data);
</script>

<form.Component {form} />
