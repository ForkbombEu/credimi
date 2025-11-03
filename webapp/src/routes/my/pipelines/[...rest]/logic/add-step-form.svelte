<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { AddStepState } from './pipeline-builder.svelte.js';

	import WalletStepFormComponent from './wallet-step-form.svelte';
	import { WalletStepForm } from './wallet.svelte.js';

	//

	let { state }: { state: AddStepState } = $props();
</script>

{#if state instanceof WalletStepForm}
	<WalletStepFormComponent form={state} />
	<!-- {:else if state instanceof AddCredentialState}
	addcredentialform
{:else if state instanceof AddCustomCheckState}
	addcustomcheckform
{:else if state instanceof AddUseCaseVerificationState}
	addusecaseverificationform -->
{/if}
