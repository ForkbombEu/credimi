<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { createForm, Form } from '@/forms';
	import { Field } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	let { data } = $props();

	const form = createForm({
		adapter: zod(
			z
				.object({
					password: z.string().min(8),
					passwordConfirmation: z.string().min(8)
				})
				.refine(
					(data) => data.password === data.passwordConfirmation,
					m.PASSWORDS_DO_NOT_MATCH()
				)
		),
		onSubmit: async ({ form }) => {
			const { password, passwordConfirmation } = form.data;
			await pb
				.collection('users')
				.confirmPasswordReset(data.token, password, passwordConfirmation);
			state = 'success';
		}
	});

	let state: 'form' | 'success' = $state('form');
</script>

{#if state === 'form'}
	<div class="space-y-1">
		<T tag="h4">{m.Reset_password()}</T>
		<T>{m.Please_enter_here_a_new_password_()}</T>
	</div>

	<Form {form}>
		<Field
			{form}
			name="password"
			options={{ type: 'password', label: m.New_password(), placeholder: '•••••••••••' }}
		/>

		<Field
			{form}
			name="passwordConfirmation"
			options={{ type: 'password', label: m.Confirm_password(), placeholder: '•••••••••••' }}
		/>

		{#snippet submitButton({ SubmitButton })}
			<SubmitButton class="w-full">{m.Reset_password()}</SubmitButton>
		{/snippet}
	</Form>
{:else if state === 'success'}
	<Alert variant="success" class="-rotate-1 space-y-4 bg-green-50">
		<T tag="h4">{m.Password_reset_successfully()}</T>
		<Button class="w-full" href="/login">{m.Go_to_login()}</Button>
	</Alert>
{/if}
