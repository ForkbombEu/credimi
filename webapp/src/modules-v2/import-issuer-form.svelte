<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { ArrowLeft, CheckCircle2, Download } from 'lucide-svelte';
	import { ClientResponseError } from 'pocketbase';
	import { fade } from 'svelte/transition';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';

	import type { CredentialIssuersResponse } from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import { Alert, AlertDescription } from '@/components/ui/alert';
	import { createForm, Form, SubmitButton } from '@/forms';
	import { Field } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	import { getExceptionMessage } from '@/utils/errors';

	import { fetchCredentialIssuer, getCredentialIssuerByUrl } from './utils';

	//

	type Props = {
		organizationId: string;
		onSuccess?: () => void;
	};

	let { organizationId, onSuccess }: Props = $props();

	//

	let importedCredentialIssuer = $state<CredentialIssuersResponse>();

	const form = createForm({
		adapter: zod(
			z.object({
				url: z.string().trim().url()
			})
		),
		onSubmit: async ({ form }) => {
			const { url } = form.data;
			await fetchCredentialIssuer(url);
			importedCredentialIssuer = await getCredentialIssuerByUrl(url, organizationId);
			// onSuccess?.();
		},
		onError: ({ error, errorMessage, setFormError }) => {
			if (error instanceof ClientResponseError) {
				if (error.response?.error?.code === 404) {
					return setFormError(m.Could_not_import_credential_issuer_well_known());
				}
				setFormError(error.response?.error?.message || errorMessage);
			} else {
				setFormError(getExceptionMessage(error));
			}
		}
	});

	async function discard() {
		if (!importedCredentialIssuer?.id) return;
		await pb.collection('credential_issuers').delete(importedCredentialIssuer.id);
		importedCredentialIssuer = undefined;
	}
</script>

{#if !importedCredentialIssuer}
	<div
		in:fade
		class="bg-secondary border-purple-outline/20 flex items-start gap-3 rounded-md p-4"
	>
		<Download class="text-secondary-foreground mt-0.5 h-5 w-5 flex-shrink-0" />
		<div>
			<h4 class="text-secondary-foreground mb-1 text-base font-medium">
				<strong>Optional</strong>: Import credential issuer
			</h4>

			<p class="text-secondary-foreground/80 mb-4 text-sm">
				Import a new credential issuer by providing its URL. This will create a new issuer
				record, fetch the issuer's well-known metadata and automatically discover available
				credentials. If the URL already exists in your organization, the existing credential
				issuer will be refrehed.
			</p>

			<Form {form} hideRequiredIndicator class="space-y-4" hide={['submit_button']}>
				<div class="flex gap-2">
					<div class="grow">
						<Field
							{form}
							name="url"
							options={{
								type: 'url',
								hideLabel: true,
								placeholder: 'https://example-issuer.org'
							}}
						/>
					</div>
					<SubmitButton variant="outline" class="flex w-fit">{m.Import()}</SubmitButton>
				</div>
			</Form>
		</div>
	</div>
{:else}
	<div in:fade>
		<Button size="sm" variant="outline" class="mb-2" onclick={discard}>
			<ArrowLeft />
			Back and discard
		</Button>

		<Alert class="mb-6 border-green-200 bg-green-50">
			<CheckCircle2 class="h-4 w-4 text-green-600" />
			<AlertDescription class="text-green-800">
				Credential issuer imported successfully!<br />
				Edit the form to add more information and help discoverability.
			</AlertDescription>
		</Alert>

		<CollectionForm
			collection="credential_issuers"
			initialData={importedCredentialIssuer}
			recordId={importedCredentialIssuer.id}
			fieldsOptions={{
				exclude: ['owner', 'imported', 'url', 'workflow_url', 'published']
			}}
			{onSuccess}
			uiOptions={{
				toastText: 'Credential issuer imported successfully!'
			}}
		/>
	</div>
{/if}
