<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { StepFormState } from './types.js';

	import BaseStepFormComponent from './base-step-form.svelte';
	import { BaseStepForm } from './base-step-form.svelte.js';
	import WalletStepFormComponent from './wallet-step-form.svelte';
	import { WalletStepForm } from './wallet-step-form.svelte.js';

	//

	let { state }: { state: StepFormState } = $props();
</script>

{#if state instanceof WalletStepForm}
	<WalletStepFormComponent form={state} />
{:else if state instanceof BaseStepForm}
	<BaseStepFormComponent form={state} />
{/if}
