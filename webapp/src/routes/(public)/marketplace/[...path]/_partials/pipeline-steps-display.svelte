<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { BuilderStep } from '$lib/pipeline-form/steps-builder/types';

	import StepCardDisplay from '$lib/pipeline-form/steps-builder/utils/step-card-display.svelte';

	//

	type Props = {
		steps: BuilderStep[];
	};

	let { steps }: Props = $props();
</script>

<div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
	{#each steps as step, index (step.id)}
		<StepCardDisplay {step} {index} />
	{/each}
</div>
