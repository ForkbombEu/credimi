<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import CredentialIssuerForm from '$services-and-products/_credentials/credential-issuer-form.svelte';
	import { CheckCircle2 } from 'lucide-svelte';

	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import WalletForm from './_partials/wallet-form.svelte';

	//

	type TestSubject = 'credential_issuer' | 'wallet';

	const testSubjectsLabels: Record<TestSubject, string> = {
		credential_issuer: 'Credential issuer',
		wallet: 'Wallet'
	};

	let currentTestSubject = $state<TestSubject>();
</script>

<div class="mx-auto max-w-xl space-y-12 px-8 py-12">
	<T tag="h1">{m.Start_a_new_compliance_check()}</T>

	<div class="space-y-2">
		<T>{m.What_do_you_want_to_check()}</T>
		<div class="flex gap-4">
			{@render TestOptionButton('credential_issuer')}
			{@render TestOptionButton('wallet')}
		</div>
	</div>

	<div>
		{#if currentTestSubject == 'credential_issuer'}
			<CredentialIssuerForm />
		{:else if currentTestSubject == 'wallet'}
			<WalletForm />
		{/if}
	</div>
</div>

{#snippet TestOptionButton(subject: TestSubject)}
	{@const select = () => {
		currentTestSubject = subject;
	}}
	{@const isSelected = currentTestSubject == subject}
	<button
		class={[
			'bg-secondary ring-primary relative flex grow basis-1 rounded-lg p-4 hover:ring-2',
			{ 'ring-2': isSelected }
		]}
		onclick={select}
	>
		{testSubjectsLabels[subject]}

		{#if isSelected}
			<CheckCircle2 class="absolute right-1 top-1" />
		{/if}
	</button>
{/snippet}
