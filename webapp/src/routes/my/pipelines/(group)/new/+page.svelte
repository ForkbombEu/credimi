<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Render } from '$lib/renderable';
	import { PipelineForm } from '$pipeline-form/pipeline-form.svelte.js';

	const form = new PipelineForm({ mode: 'create' });
</script>

<Render item={form} />
