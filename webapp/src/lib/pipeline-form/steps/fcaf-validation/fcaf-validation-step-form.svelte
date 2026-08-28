<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { WithLabel } from '$pipeline-form/steps/_partials/index.js';

	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { m } from '@/i18n';

	import type { FCAFValidationStepForm } from './fcaf-validation-step-form.svelte.js';

	//

	let { self: form }: SelfProp<FCAFValidationStepForm> = $props();
</script>

<div class="space-y-6 p-4">
	<WithLabel label="FCAF validation configuration" required>
		<CodeEditor lang="yaml" bind:value={form.data.yaml} />
	</WithLabel>

	{#if form.validationError}
		<p class="text-sm text-destructive">{form.validationError}</p>
	{/if}

	{#if form.intent === 'add'}
		<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
			<T>{m.Add_step()}</T>
		</Button>
	{/if}
</div>
