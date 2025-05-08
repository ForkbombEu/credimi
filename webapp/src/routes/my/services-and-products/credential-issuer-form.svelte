<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createForm, Form, SubmitButton } from '@/forms';
	import { Field } from '@/forms/fields';
	import { m } from '@/i18n';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';
	import { pb } from '@/pocketbase';

	//

	type Props = {
		onSuccess?: () => void;
	};

	let { onSuccess }: Props = $props();

	const form = createForm({
		adapter: zod(
			z.object({
				url: z.string().trim().url()
			})
		),
		onSubmit: async ({ form }) => {
			// note: use npm:out-of-character to clean the url if needed
			const { url } = form.data;

			await pb.send('/credentials_issuers/start-check', {
				method: 'POST',
				body: {
					credentialIssuerUrl: url
				}
			});

			onSuccess?.();
		}
	});
</script>

<Form {form} hideRequiredIndicator>
	<Field {form} name="url" options={{ type: 'url', label: m.Credential_issuer_URL() }} />

	{#snippet submitButton()}
		<SubmitButton class="flex w-full">{m.Add_new_credential_issuer()}</SubmitButton>
	{/snippet}
</Form>
