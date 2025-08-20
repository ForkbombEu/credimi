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
	import { Loader2, Download, AlertCircle } from 'lucide-svelte';
	import { Button } from '@/components/ui/button';
	import {
		Card,
		CardContent,
		CardDescription,
		CardHeader,
		CardTitle
	} from '@/components/ui/card';
	import { Alert, AlertDescription } from '@/components/ui/alert';
	import Separator from '@/components/ui/separator/separator.svelte';

	//

	type Props = {
		onSuccess?: () => void;
		initialData?: Partial<WalletsResponse>;
		walletId?: string;
	};

	let { onSuccess, initialData = {}, walletId }: Props = $props();

	//

	let isProcessingWorkflow = $state(false);
	let autoPopulateUrl = $state('');
	let autoPopulateError = $state('');

	const schema = createCollectionZodSchema('wallets')
		.omit({
			owner: true
		})
		.extend({
			conformance_checks: z.array(ConformanceCheckSchema).nullable()
		});

	// Function to auto-populate form from store URLs
	async function autoPopulateFromUrl() {
		if (!autoPopulateUrl.trim()) return;

		isProcessingWorkflow = true;
		autoPopulateError = '';

		try {
			const response = await pb.send('/wallet/start-check', {
				method: 'POST',
				body: {
					WalletURL: autoPopulateUrl.trim()
				}
			});

			// Auto-populate the form with the response data
			if (response) {
				// Update the form data with the fetched metadata
				const { form: formData } = editWalletform;
				formData.update((currentData) => {
					const updatedData = { ...currentData };
					Object.keys(response).forEach((key) => {
						if (key in updatedData) {
							(updatedData as any)[key] = response[key];
						}
					});
					return updatedData;
				});
			}

			// Clear the URL input after successful fetch
			autoPopulateUrl = '';
		} catch (error: any) {
			if (error?.response?.error?.code === 404) {
				autoPopulateError = m.Could_not_import_credential_issuer_well_known();
			} else {
				autoPopulateError =
					error?.response?.error?.message || 'Failed to fetch wallet metadata';
			}
		} finally {
			isProcessingWorkflow = false;
		}
	}

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
	});

	// Check if either app store URL is filled to show auto-populate option
	const { form: formData } = editWalletform;
	const hasStoreUrl = $derived($formData.playstore_url || $formData.appstore_url);
</script>

<!-- Auto-populate section -->

<div class="flex flex-col">
	<div class="flex items-center mb-2">
		<!-- <Download class="h-5 w-5 mr-4" /> -->
		<T tag="h4">Import from marketplace</T>
	</div>
	<T tag="small">
		 If your wallet is already published, you can import its metadata and later edit it
	</T>
</div>

{#if autoPopulateError}
	<Alert variant="destructive">
		<AlertCircle class="h-4 w-4" />
		<AlertDescription>{autoPopulateError}</AlertDescription>
	</Alert>
{/if}

<div class="flex gap-2">
	<div class="flex-1">
		<input
			type="url"
			bind:value={autoPopulateUrl}
			placeholder="https://apps.apple.com/app/... or https://play.google.com/store/apps/..."
			class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-10 w-full rounded-md border px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
		/>
	</div>
	<Button
		type="button"
		variant="outline"
		onclick={autoPopulateFromUrl}
		disabled={isProcessingWorkflow || !autoPopulateUrl.trim()}
	>
		{#if isProcessingWorkflow}
			<Loader2 class="mr-2 h-4 w-4 animate-spin" />
			{m.Processing_wallet() || 'Fetching...'}
		{:else}
			<Download class="mr-2 h-4 w-4" />
			Import
		{/if}
	</Button>
</div>

<Separator/>
<!-- Main wallet form -->
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
	<!-- @ts-ignore -->
	<!-- TODO - Typecheck -->
	<Table
		form={editWalletform as any}
		name="conformance_checks"
		options={{ label: m.Conformance_Checks() }}
	/>
</Form>
