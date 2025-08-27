<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createForm, Form } from '@/forms';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { Field } from '@/forms/fields';
	import type { CredentialIssuersResponse } from '@/pocketbase/types';
	import { fromStore } from 'svelte/store';
	import type { Snippet } from 'svelte';

	//

	type Props = {
		organizationId: string;
		after?: Snippet<[{ formState: typeof formState }]>;
	};

	let { organizationId, after }: Props = $props();

	//

	const form = createForm({
		adapter: zod(
			z.object({
				url: z.string().trim().url()
			})
		),
		onError: ({ error, errorMessage, setFormError }) => {
			//@ts-ignore
			if (error.response?.error?.code === 404) {
				return setFormError(m.Could_not_import_credential_issuer_well_known());
			}
			//@ts-ignore
			setFormError(error.response?.error?.message || errorMessage);
		},
		onSubmit: async ({ form }) => {
			const { url } = form.data;

			await pb.send('/credentials_issuers/start-check', {
				method: 'POST',
				body: {
					credentialIssuerUrl: url
				}
			});

			credentialIssuer = await getCreatedCredentialIssuer(url);
		}
	});

	async function getCreatedCredentialIssuer(url: string) {
		const record = await pb.collection('credential_issuers').getFullList({
			filter: `owner = "${organizationId}" && url = "${url}"`
		});
		if (record.length != 1) throw new Error('Unexpected number of records');

		return record[0];
	}

	//

	const loadingState = fromStore(form.delayed);
	let credentialIssuer = $state<CredentialIssuersResponse>();

	const formState = $derived({
		loading: loadingState.current,
		credentialIssuer: credentialIssuer
	});
</script>

<Form {form}>
	<Field {form} name="url" options={{ type: 'url', label: m.Credential_issuer_URL() }} />
</Form>

{@render after?.({ formState })}
