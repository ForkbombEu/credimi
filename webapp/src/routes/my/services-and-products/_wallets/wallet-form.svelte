<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { createForm, Form, SubmitButton } from '@/forms';
	import { Field, FileField } from '@/forms/fields';
	import { pb } from '@/pocketbase/index.js';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';
	import Table, { ConformanceCheckSchema } from './wallet-form-checks-table.svelte';
	import type { WalletsResponse } from '@/pocketbase/types';
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';
	import _ from 'lodash';
	import { m } from '@/i18n';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Loader2 } from 'lucide-svelte';

	//

	type Props = {
		onSuccess?: () => void;
		initialData?: Partial<WalletsResponse>;
		walletId?: string;
	};

	let { onSuccess, initialData = {}, walletId }: Props = $props();

	//

	let isProcessingWorkflow = $state(false);
	
	const schema = createCollectionZodSchema('wallets')
		.omit({
			owner: true
		})
		.extend({
			conformance_checks: z.array(ConformanceCheckSchema).nullable()
		});

	const newWalletForm = createForm({
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

			isProcessingWorkflow = true;
			await pb.send('/wallet/start-check', {
				method: 'POST',
				body: {
					WalletURL: url
				}
			});
			isProcessingWorkflow = false;
			onSuccess?.();
		}
	});

	const editWalletform = createForm<z.infer<typeof schema>>({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			let wallet: WalletsResponse;
			if (walletId) {
				// Temp fix
				const data = _.omit(form.data, 'conformance_checks');
				wallet = await pb.collection('wallets').update(walletId, data);
			} else {
				wallet = await pb.collection('wallets').create(form.data);
			}
			onSuccess?.();
		},
		options: {
			dataType: 'form'
		},
		initialData: {
			name: initialData.name || '',
			description: initialData.description || '',
			playstore_url: initialData.playstore_url || '',
			appstore_url: initialData.appstore_url || '',
			repository: initialData.repository || '',
			home_url: initialData.home_url || '',
			app_id: initialData.app_id || '',
			logo_url: initialData.logo_url || '',
			// Don't include apk/logo in initial data since they're File fields
			conformance_checks: null
		}
		// TODO - Fix edit form for conformance_checks
		// {
		// 	..._.omit(initialData, 'logo', "conformance_checks"),
		// 	// conformance_checks: initialData.conformance_checks as NonEmptyArray<ConformanceCheck>
		// }
	});
</script>

{#if !walletId}
	{#if isProcessingWorkflow}
		<!-- Spinner state while workflow is processing -->
		   <div class="flex flex-col items-center justify-center space-y-4 py-8">
			   <Loader2 class="text-primary h-8 w-8 animate-spin" />
			   <T class="text-muted-foreground text-center">
				   {m.Processing_wallet()}<br />
				   <span class="text-sm">{m.Fetching_app_store_metadata()}</span>
			   </T>
		   </div>
	{:else}
		<!-- Normal form state -->
		<Form form={newWalletForm} hideRequiredIndicator>
			   <Field
				   form={newWalletForm}
				   name="url"
				   options={{
					   type: 'url',
					   label: m.Wallet_URL(),
					   placeholder: m.Enter_wallet_URL()
				   }}
			   />

			{#snippet submitButton()}
				<SubmitButton class="flex w-full">{m.Add_new_wallet()}</SubmitButton>
			{/snippet}
		</Form>
	{/if}
{:else}
	<Form form={editWalletform} enctype="multipart/form-data" class="!space-y-8">
		   <Field
			   form={editWalletform}
			   name="name"
			   options={{
				   type: 'text',
				   label: m.App_Name(),
				   placeholder: m.Enter_app_name()
			   }}
		   />
		<MarkdownField form={editWalletform} name="description" height={80} />
		   <Field
			   form={editWalletform}
			   name="logo_url"
			   options={{
				   type: 'url',
				   label: m.Logo_URL(),
				   placeholder: m.Enter_logo_URL()
			   }}
		   />
		   <Field
			   form={editWalletform}
			   name="app_id"
			   options={{
				   type: 'text',
				   label: m.App_ID(),
				   placeholder: m.Enter_app_identifier()
			   }}
		   />
		   <Field
			   form={editWalletform}
			   name="playstore_url"
			   options={{
				   type: 'url',
				   label: m.Play_Store_URL(),
				   placeholder: m.Enter_Play_Store_URL()
			   }}
		   />
		   <Field
			   form={editWalletform}
			   name="appstore_url"
			   options={{
				   type: 'url',
				   label: m.App_Store_URL(),
				   placeholder: m.Enter_App_Store_URL()
			   }}
		   />
		   <Field
			   form={editWalletform}
			   name="repository"
			   options={{
				   type: 'url',
				   label: m.Repository_URL(),
				   placeholder: m.Enter_repository_URL()
			   }}
		   />
		   <Field
			   form={editWalletform}
			   name="home_url"
			   options={{
				   type: 'url',
				   label: m.Home_URL(),
				   placeholder: m.Enter_home_URL()
			   }}
		   />
		<!-- {walletId}
		{#if !walletId} -->
		<!-- @ts-ignore -->
		<!-- TODO - Typecheck -->
		   <Table
			   form={editWalletform as any}
			   name="conformance_checks"
			   options={{ label: m.Conformance_Checks() }}
		   />
		<!-- {:else}
			<Alert variant="info" icon={InfoIcon}>
				<T>Editing conformance checks for wallets is temporary disabled.</T>
			</Alert>
		{/if} -->
	</Form>
{/if}
