<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CircleAlert, Download, Info, Loader } from '@lucide/svelte';

	import type { WalletsResponse } from '@/pocketbase/types';

	import { setupCollectionForm } from '@/collections-components/form/collectionFormSetup';
	import { Alert, AlertDescription } from '@/components/ui/alert';
	import { Button } from '@/components/ui/button';
	import { Card, CardContent } from '@/components/ui/card';
	import Input from '@/components/ui/input/input.svelte';
	import Separator from '@/components/ui/separator/separator.svelte';
	import { Form } from '@/forms';
	import { Field } from '@/forms/fields';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase/index.js';

	import LogoField from './logo-field.svelte';

	//

	type Props = {
		onSuccess?: () => void;
		walletId?: string;
		initialData?: WalletsResponse;
	};

	let { onSuccess, initialData, walletId }: Props = $props();

	//

	const editWalletform = setupCollectionForm({
		collection: 'wallets',
		recordId: walletId,
		fieldsOptions: {
			exclude: ['owner', 'conformance_checks']
		},
		initialData: initialData,
		onSuccess: onSuccess
	});

	//

	let isProcessingWorkflow = $state(false);
	let autoPopulateUrl = $state('');
	let autoPopulateError = $state('');

	async function autoPopulateFromUrl() {
		if (!autoPopulateUrl.trim()) return;

		isProcessingWorkflow = true;
		autoPopulateError = '';

		try {
			const response = await pb.send('api/wallet/start-check', {
				method: 'POST',
				body: {
					WalletURL: autoPopulateUrl.trim()
				}
			});
			if (!response) return;

			if (response.logo) {
				response.logo_url = response.logo;
				delete response.logo;
			}

			const { form: formDataStore } = editWalletform;
			formDataStore.update((currentData) => {
				const updatedData = { ...currentData };
				Object.keys(response).forEach((key) => {
					if (key in updatedData && key in response) {
						(updatedData as Record<string, unknown>)[key] =
							response[key as keyof typeof response];
					}
				});
				return updatedData;
			});

			autoPopulateUrl = '';
		} catch (error: unknown) {
			const errorObj = error as ErrorResponse;
			if (errorObj?.response?.error?.code === 404) {
				autoPopulateError = m.Wallet_not_found_check_URL();
			} else {
				autoPopulateError =
					errorObj?.response?.error?.message || m.Failed_to_fetch_wallet_metadata();
			}
		}

		isProcessingWorkflow = false;
	}

	type ErrorResponse = {
		response?: { error?: { code?: number; message?: string } };
	};
</script>

<Card class="bg-secondary border-purple-200 shadow-none! mb-8 py-0">
	<CardContent class="space-y-4 p-6">
		<div class="flex items-start gap-3">
			<Info class="text-secondary-foreground mt-0.5 h-5 w-5 shrink-0" />
			<div class="flex-1 space-y-1">
				<h4 class="text-secondary-foreground text-base font-medium">
					{m.Import_from_marketplace()}
				</h4>
				<p class="text-secondary-foreground/80 text-sm">
					{m.Import_wallet_metadata_description()}
				</p>
			</div>
		</div>

		{#if autoPopulateError}
			<Alert variant="destructive">
				<CircleAlert class="h-4 w-4" />
				<AlertDescription>{autoPopulateError}</AlertDescription>
			</Alert>
		{/if}

		<div class="flex gap-2">
			<div class="flex-1">
				<Input
					type="url"
					bind:value={autoPopulateUrl}
					placeholder={m.Enter_app_store_URL_placeholder()}
				/>
			</div>
			<Button
				type="button"
				variant="outline"
				onclick={autoPopulateFromUrl}
				disabled={isProcessingWorkflow || !autoPopulateUrl.trim()}
			>
				{#if isProcessingWorkflow}
					<Loader class="animate-spin" />
					{m.Processing_wallet() || m.Fetching_fallback()}
				{:else}
					<Download />
					{m.Import()}
				{/if}
			</Button>
		</div>
	</CardContent>
</Card>

<!-- Main Form Section -->

<div class="space-y-6">
	<div class="flex items-center gap-2">
		<h3 class="text-lg font-semibold">Wallet Details</h3>
		<Separator class="flex-1" />
	</div>

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

		<MarkdownField form={editWalletform} name="description" />

		<LogoField form={editWalletform} walletResponse={initialData} />

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
	</Form>
</div>
