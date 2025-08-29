<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import _ from 'lodash';
	import { AlertCircle, Download, Info, Loader2, X } from 'lucide-svelte';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';

	import type { WalletsResponse } from '@/pocketbase/types';

	import { Alert, AlertDescription } from '@/components/ui/alert';
	import { Button } from '@/components/ui/button';
	import { Card, CardContent } from '@/components/ui/card';
	import Separator from '@/components/ui/separator/separator.svelte';
	import { createForm, Form } from '@/forms';
	import { Field, FileField } from '@/forms/fields';
	import MarkdownField from '@/forms/fields/markdownField.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase/index.js';
	import { createCollectionZodSchema } from '@/pocketbase/zod-schema';

	import Table, { ConformanceCheckSchema } from './wallet-form-checks-table.svelte';
	/** eslint-disable @typescript-eslint/no-explicit-any */

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
	let logoUrlError = $state('');

	const schema = createCollectionZodSchema('wallets')
		.omit({
			owner: true
		})
		.extend({
			conformance_checks: z.array(ConformanceCheckSchema).nullable()
		});

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

			if (response) {
				if (response.logo) {
					response.logo_url = response.logo;
					delete response.logo;
				}
				const { form: formDataStore } = editWalletform;
				formDataStore.update((currentData) => {
					const updatedData = { ...currentData };
					Object.keys(response).forEach((key) => {
						if (key in updatedData) {
							// eslint-disable-next-line @typescript-eslint/no-explicit-any
							(updatedData as any)[key] = response[key];
						}
					});
					return updatedData;
				});
			}

			autoPopulateUrl = '';
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
		} catch (error: any) {
			if (error?.response?.error?.code === 404) {
				autoPopulateError = m.Wallet_not_found_check_URL();
			} else {
				autoPopulateError =
					error?.response?.error?.message || m.Failed_to_fetch_wallet_metadata();
			}
		} finally {
			isProcessingWorkflow = false;
		}
	}

	function removeLogo() {
		const { form: formDataStore } = editWalletform;
		formDataStore.update((currentData) => ({
			...currentData,
			logo: undefined,
			logo_url: ''
		}));
		logoUrlError = '';
	}

	const editWalletform = createForm<z.infer<typeof schema>>({
		adapter: zod(schema),
		onSubmit: async ({ form }) => {
			const formData = { ...form.data };
			if (!formData.apk || (formData.apk instanceof File && formData.apk.size === 0)) {
				delete formData.apk;
			}
			if (!formData.logo || (formData.logo instanceof File && formData.logo.size === 0)) {
				delete formData.logo;
			}

			if (walletId) {
				// Temp fix
				const data = _.omit(formData, 'conformance_checks');
				await pb.collection('wallets').update(walletId, data);
			} else {
				await pb.collection('wallets').create(formData);
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
			logo_url: initialData.logo_url || '',
			// Don't include apk/logo in initial data since they're File fields
			conformance_checks: null
		}
	});

	const { form: formData } = editWalletform;
</script>

<Card class="bg-muted border-muted/40 mb-8">
	<CardContent class="space-y-4 p-6">
		<div class="flex items-start gap-3">
			<Info class="text-muted-foreground mt-0.5 h-5 w-5 flex-shrink-0" />
			<div class="flex-1 space-y-1">
				<h4 class="text-base font-medium">{m.Import_from_marketplace()}</h4>
				<p class="text-muted-foreground text-sm">
					{m.Import_wallet_metadata_description()}
				</p>
			</div>
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
					placeholder={m.Enter_app_store_URL_placeholder()}
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
					{m.Processing_wallet() || m.Fetching_fallback()}
				{:else}
					<Download class="mr-2 h-4 w-4" />
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
		<MarkdownField form={editWalletform} name="description" height={80} />
		<div class="space-y-4">
			<div class="text-sm font-medium leading-none">{m.Logo()}</div>
			{#if $formData.logo instanceof File && $formData.logo.size > 0 && !logoUrlError}
				<div class="relative mb-2 inline-block">
					<img
						src={URL.createObjectURL($formData.logo)}
						alt={m.Logo_preview()}
						class="max-h-32 rounded border"
					/>
					<Button
						size="sm"
						variant="destructive"
						class="absolute -right-2 -top-2 h-6 w-6 rounded-full p-0"
						onclick={removeLogo}
					>
						<X class="h-4 w-4" />
					</Button>
				</div>
			{:else if $formData.logo_url && !logoUrlError}
				<div class="relative mb-2 inline-block">
					<img
						src={$formData.logo_url}
						alt={m.Logo_preview()}
						class="max-h-32 rounded border"
						onerror={() => {
							logoUrlError = m.Invalid_image_URL_error();
						}}
						onload={() => {
							logoUrlError = '';
						}}
					/>
					<Button
						size="sm"
						variant="destructive"
						class="absolute -right-2 -top-2 h-6 w-6 rounded-full p-0"
						onclick={removeLogo}
					>
						<X class="h-4 w-4" />
					</Button>
				</div>
			{/if}

			{#if logoUrlError}
				<Alert variant="destructive">
					<AlertCircle class="h-4 w-4" />
					<AlertDescription>{logoUrlError}</AlertDescription>
				</Alert>
			{/if}

			{#if (!($formData.logo instanceof File && $formData.logo.size > 0) && !$formData.logo_url) || logoUrlError}
				<div class="space-y-4">
					<div class="space-y-2">
						<FileField
							form={editWalletform}
							name="logo"
							options={{
								label: '',
								placeholder: m.Upload_logo(),
								showFilesList: false
							}}
						/>
					</div>
					<div class="relative">
						<div class="absolute inset-0 flex items-center">
							<span class="border-muted w-full border-t"></span>
						</div>
						<div class="relative flex justify-center text-xs uppercase">
							<span class="bg-background text-muted-foreground px-2">{m.or()}</span>
						</div>
					</div>
					<div class="space-y-2">
						<Field
							form={editWalletform}
							name="logo_url"
							options={{
								type: 'url',
								label: '',
								placeholder: m.Enter_logo_URL()
							}}
						/>
					</div>
				</div>
			{/if}
		</div>
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
			form={editWalletform}
			name="conformance_checks"
			options={{ label: m.Conformance_Checks() }}
		/>
	</Form>
</div>
