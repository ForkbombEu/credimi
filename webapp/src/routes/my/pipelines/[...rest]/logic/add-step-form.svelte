<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import {
		AddCredentialState,
		AddCustomCheckState,
		AddUseCaseVerificationState,
		AddWalletState,
		type AddStepState
	} from './pipeline-builder.svelte.js';

	//

	let { state }: { state: AddStepState } = $props();
</script>

{#if state instanceof AddWalletState}
	addwalletform
{:else if state instanceof AddCredentialState}
	addcredentialform
{:else if state instanceof AddCustomCheckState}
	addcustomcheckform
{:else if state instanceof AddUseCaseVerificationState}
	addusecaseverificationform
{/if}
