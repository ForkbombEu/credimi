<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="C extends CollectionName">
	import { createForm, Form, SubmitButton } from '@/forms';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { m } from '@/i18n';
	import { type CollectionFormOptions } from '@/collections-components/form/collectionFormTypes';
	import type { CollectionName } from '@/pocketbase/collections-models';
	import Button from '@/components/ui-custom/button.svelte';
	import { getCollectionManagerContext } from '../collectionManagerContext';
	import { CollectionForm } from '@/collections-components';
	import { Plus } from 'lucide-svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import { merge } from 'lodash';
	import type { RecordCreateEditProps } from './types';
	import { pb } from '@/pocketbase';
	import { currentUser } from '@/pocketbase';
	import { Field } from '@/forms/fields';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';

	//

	const {
		formTitle,
		onSuccess = () => {},
		buttonText,
		button
	}: RecordCreateEditProps<C> = $props();

	const { manager, formsOptions } = $derived(getCollectionManagerContext());

	const defaultFormOptions: CollectionFormOptions<C> = {
		uiOptions: { showToastOnSuccess: true }
	};
	const options = $derived(merge(defaultFormOptions, formsOptions.base, formsOptions.edit));

	const sheetTitle = $derived(formTitle ?? m.Create_record());
	console.log('RECORD CREATE TWO OPTIONS:');
	$inspect(options);

	// State for managing the created credential issuer
	let credentialIssuerResponse: { credentialIssuerUrl: string; workflowUrl: string } | null = $state(null);
	let createdRecord: any = $state(null);
	let isLoadingRecord = $state(false);
	let hasAttemptedSubmit = $state(false);
	let lastSubmittedUrl = $state('');
	let hasBeenSaved = $state(false);
	let isSheetOpen = $state(false);

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
			// note: use npm:out-of-character to clean the url if needed
			const { url } = form.data;
			hasAttemptedSubmit = true;
			lastSubmittedUrl = url;
			
			try {
				const response = await pb.send('/credentials_issuers/start-check', {
					method: 'POST',
					body: {
						credentialIssuerUrl: url
					}
				});
				console.log("RESPONSE SUCCESSFUL");
				console.log(response);
				
				// Save the response
				credentialIssuerResponse = response;
				
				// Small delay to ensure the record is fully created on the backend
				await new Promise(resolve => setTimeout(resolve, 500));
				
				// Fetch the created record
				await loadCreatedRecord(url);
			} catch (error) {
				throw error; // Let the form handle the error
			}
		}
	});

	// Auto-submit when URL is valid
	const { form: formData, errors, submitting } = form;
	$effect(() => {
		const url = $formData.url;
		const urlErrors = $errors.url;
		
		// Only auto-submit if:
		// 1. URL is valid
		// 2. No errors on URL field
		// 3. Not currently submitting
		// 4. Haven't already attempted this URL
		// 5. No response yet (success or error)
		if (url && 
			!urlErrors?.length && 
			!$submitting && 
			url !== lastSubmittedUrl &&
			!credentialIssuerResponse) {
			
			// Small delay to ensure the validation is complete
			setTimeout(() => {
				if (!urlErrors?.length && 
					!$submitting && 
					url === $formData.url && // Make sure URL hasn't changed
					url !== lastSubmittedUrl) {
					
					// Trigger form submission by creating a submit event
					const submitEvent = new Event('submit', { bubbles: true, cancelable: true });
					const formElement = document.querySelector('form');
					formElement?.dispatchEvent(submitEvent);
				}
			}, 100);
		}
	});

	async function loadCreatedRecord(url: string) {
		console.log("LOADING CREATED RECORD");
		
		if (!credentialIssuerResponse) return;
		
		isLoadingRecord = true;
		try {
			// Find the record by URL and current user's organization
			console.log("Getting record...");
			
			// Get the current user's organization through orgAuthorizations
			if (!$currentUser?.id) {
				console.error('No current user found');
				return;
			}

			const orgAuth = await pb.collection('orgAuthorizations').getFirstListItem(
				`user.id = "${$currentUser.id}"`,
				{ expand: 'organization' }
			);

			const organization = (orgAuth.expand as any)?.organization;
			if (!organization?.id) {
				console.error('No organization found for user');
				return;
			}
			
			const records = await pb.collection('credential_issuers').getList(1, 1, {
				filter: `url = "${credentialIssuerResponse.credentialIssuerUrl}" && owner = "${organization.id}"`
			});

			console.log("RESPONSE:");
			console.log(records);
			
			
			if (records.items.length > 0) {
				console.log("Setting createdRecord:", records.items[0]);
				createdRecord = records.items[0];
				console.log("createdRecord set, current URL:", $formData.url);
			} else {
				console.log("No records found for the query");
			}
		} catch (error) {
			console.error('Failed to load created record:', error);
		} finally {
			isLoadingRecord = false;
		}
	}

	async function deleteCredentialIssuer() {
		if (!createdRecord) return;
		
		try {
			await pb.collection('credential_issuers').delete(createdRecord.id);
			
			// Reset state
			credentialIssuerResponse = null;
			createdRecord = null;
			hasAttemptedSubmit = false;
			lastSubmittedUrl = '';
			hasBeenSaved = false;
			formData.set({ url: '' });
			
			// Refresh the manager
			manager.loadRecords();
		} catch (error) {
			console.error('Failed to delete credential issuer:', error);
		}
	}

	// Function to handle sheet closing
	async function handleSheetClose() {
		// If we have a created record but it hasn't been saved through the form, delete it
		if (createdRecord && !hasBeenSaved) {
			await deleteCredentialIssuer();
		}
	}

	// Watch for sheet close events
	$effect(() => {
		// When sheet closes and we have an unsaved record, clean it up
		if (!isSheetOpen && createdRecord && !hasBeenSaved) {
			handleSheetClose();
		}
	});

	// Reset state when URL changes to allow re-trying (but only if user manually changed it)
	$effect(() => {
		const url = $formData.url;
		console.log("Reset effect triggered - URL:", url, "lastSubmittedUrl:", lastSubmittedUrl);
		
		// Only reset if:
		// 1. There was a previous submission
		// 2. The current URL is different from what was submitted
		// 3. The URL change was not caused by our own reset (when deleting)
		// 4. There's already a response or record (meaning we completed a flow)
		if (lastSubmittedUrl !== '' && 
			url !== lastSubmittedUrl && 
			url !== '' && // Don't reset when we clear the form ourselves
			(credentialIssuerResponse || createdRecord)) {
			
			console.log("Resetting state due to URL change");
			hasAttemptedSubmit = false;
			credentialIssuerResponse = null;
			createdRecord = null;
			hasBeenSaved = false;
		}
	});
</script>

<Sheet title={sheetTitle} bind:open={isSheetOpen}>
	{#snippet trigger({ sheetTriggerAttributes, openSheet })}
		{#if button}
			{@render button({
				triggerAttributes: sheetTriggerAttributes,
				icon: Plus,
				openForm: openSheet
			})}
		{:else}
			<Button {...sheetTriggerAttributes} class="shrink-0">
				<Icon src={Plus} />
				{@render SubmitButtonText()}
			</Button>
		{/if}
	{/snippet}

	{#snippet content({ closeSheet })}
		<div class="space-y-4">
			<!-- URL Input Form -->
			<Form {form} hideRequiredIndicator hide={['submit_button']}>
				<Field {form} name="url" options={{ type: 'url', label: m.Credential_issuer_URL() }} />
			</Form>

			<!-- Show status and actions based on state -->
			{#if credentialIssuerResponse}
				<div class="space-y-3">
					<!-- Success message -->
					<div class="rounded-md bg-green-50 p-3">
						<p class="text-sm text-green-800">
							{`Found. ${m.Add_new_credential_issuer()}`}
						</p>
						<!-- {#if credentialIssuerResponse.workflowUrl}
							<a 
								href={credentialIssuerResponse.workflowUrl} 
								target="_blank" 
								class="text-sm text-blue-600 hover:underline"
							>
								View workflow progress
							</a>
						{/if} -->
					</div>

					<!-- Delete button for testing -->
					<!-- <div class="flex justify-between">
						<Button 
							variant="destructive" 
							size="sm" 
							onclick={deleteCredentialIssuer}
						>
							Delete test issuer
						</Button>
						<Button onclick={closeSheet}>
							Close
						</Button>
					</div> -->
				</div>
			{:else if isLoadingRecord}
				<div class="text-center">
					<p class="text-sm text-gray-600">Loading...</p>
				</div>
			{/if}

			<!-- Collection Form for the created record -->
			{#if createdRecord}
				<CollectionForm
					collection={manager.collection}
					recordId={createdRecord.id}
					initialData={createdRecord}
					{...options}
					onSuccess={(record) => {
						hasBeenSaved = true; // Mark as saved before closing
						closeSheet();
						manager.loadRecords();
						onSuccess(record, 'create');
					}}
				>
					{#snippet submitButtonContent()}
						{@render SubmitButtonText()}
					{/snippet}
				</CollectionForm>
			{/if}
		</div>
	{/snippet}
</Sheet>

{#snippet SubmitButtonText()}
	{#if buttonText}
		{@render buttonText?.()}
	{:else}
		{m.Create_record()}
	{/if}
{/snippet}
