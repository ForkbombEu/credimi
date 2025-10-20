<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeftIcon } from 'lucide-svelte';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';

	import A from '@/components/ui-custom/a.svelte';
	import Alert from '@/components/ui-custom/alert.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { createForm, Form } from '@/forms';
	import { Field } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	let state: 'form' | 'success' = $state('form');

	const form = createForm({
		adapter: zod(
			z.object({
				email: z.string().email()
			})
		),
		onSubmit: async ({ form }) => {
			await pb.collection('users').requestPasswordReset(form.data.email);
			state = 'success';
		}
	});
</script>

{#if state === 'form'}
	<A href="/login" class="flex items-center gap-1 pb-4 text-sm">
		<ArrowLeftIcon size={16} />
		<span>{m.Back()}</span>
	</A>

	<div class="space-y-1">
		<T tag="h4">{m.Forgot_password()}</T>
		<T>{m.forgot_password_description()}</T>
	</div>

	<Form {form} class="space-y-4" hideRequiredIndicator>
		<Field
			{form}
			name="email"
			options={{ type: 'email', label: m.Your_email(), placeholder: 'name@example.org' }}
		/>

		{#snippet submitButton({ SubmitButton })}
			<SubmitButton class="w-full">{m.Recover_password()}</SubmitButton>
		{/snippet}
	</Form>
{:else if state === 'success'}
	<Alert variant="success" class="-rotate-1 space-y-2 bg-green-50">
		<T tag="h4">{m.Password_reset_email_sent_successfully()}</T>
		<T>{m.Please_click_the_link_in_the_email_to_reset_your_password()}</T>
	</Alert>
{/if}
