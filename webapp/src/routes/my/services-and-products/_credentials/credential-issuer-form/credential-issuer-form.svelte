<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Plus, Trash, X } from 'lucide-svelte';

	import type { CredentialIssuersResponse } from '@/pocketbase/types';

	import { CollectionForm } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as AlertDialog from '@/components/ui/alert-dialog';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import ImportCredentialIssuer from './import-credential-issuer.svelte';

	//

	type Props = {
		organizationId: string;
		currentIssuers?: CredentialIssuersResponse[];
	};

	let { organizationId, currentIssuers = [] }: Props = $props();

	//

	let isSheetOpen = $state(false);
	let importedIssuerId = $state<string | undefined>();
	let hasBeenSaved = $state(false);
	let showCleanupDialog = $state(false);

	async function handleCleanup() {
		if (importedIssuerId && !hasBeenSaved) {
			await pb.collection('credential_issuers').delete(importedIssuerId);
		}
		importedIssuerId = undefined;
		hasBeenSaved = false;
		showCleanupDialog = false;
	}

	function confirmCleanup() {
		handleCleanup();
	}

	function cancelCleanup() {
		importedIssuerId = undefined;
		hasBeenSaved = false;
		showCleanupDialog = false;
	}

	$effect(() => {
		if (!isSheetOpen && importedIssuerId && !hasBeenSaved) {
			showCleanupDialog = true;
		}
	});
</script>

<Sheet bind:open={isSheetOpen}>
	{#snippet trigger({ sheetTriggerAttributes })}
		<Button variant={'default'} size="sm" {...sheetTriggerAttributes}>
			<Plus />
			{m.Add_new_credential_issuer()}
		</Button>
	{/snippet}

	{#snippet content({ closeSheet })}
		<div class="space-y-6">
			<T tag="h3">Add new credential issuer</T>

			<ImportCredentialIssuer
				{organizationId}
				{currentIssuers}
				onImported={(issuerId) => {
					importedIssuerId = issuerId;
				}}
			>
				{#snippet after({ formState: { loading, credentialIssuer, importMode } })}
					{#if loading}
						<div
							class={[
								'rounded-lg border bg-slate-200 py-10 text-center',
								'flex items-center justify-center gap-2',
								'animate-pulse'
							]}
						>
							<Spinner size={16} />
							<T class="text-muted-foreground">
								{m.Loading_workflow_data_may_take_some_seconds()}
							</T>
						</div>
					{:else if importMode === true}
						<!-- Show form for import mode -->
						<div class="space-y-4">
							<div class="text-muted-foreground text-sm">
								{#if importedIssuerId}
									Review and edit the imported credential issuer details below:
								{:else}
									After importing, you can review and edit the credential issuer
									details below:
								{/if}
							</div>
							<div
								class:pointer-events-none={!importedIssuerId}
								class:opacity-50={!importedIssuerId}
							>
								<CollectionForm
									collection="credential_issuers"
									recordId={importedIssuerId}
									initialData={credentialIssuer}
									fieldsOptions={{
										exclude: [
											'published',
											'owner',
											'url',
											'canonified_name',
											'imported'
										]
									}}
									onSuccess={async () => {
										hasBeenSaved = true;
										closeSheet();
									}}
									uiOptions={{
										showToastOnSuccess: true
									}}
								></CollectionForm>
							</div>
						</div>
					{:else}
						<!-- Manual form -->
						<CollectionForm
							collection="credential_issuers"
							fieldsOptions={{
								hide: {
									owner: organizationId
								},
								exclude: ['published', 'imported', 'canonified_name']
							}}
							onSuccess={async () => {
								hasBeenSaved = true;
								closeSheet();
							}}
							uiOptions={{
								showToastOnSuccess: true
							}}
						></CollectionForm>
					{/if}
				{/snippet}
			</ImportCredentialIssuer>
		</div>
	{/snippet}
</Sheet>

<!-- Cleanup Confirmation Dialog -->
<AlertDialog.Root bind:open={showCleanupDialog}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Delete imported credential issuer?</AlertDialog.Title>
			<AlertDialog.Description>
				You have imported a credential issuer but haven't saved it yet. Would you like to
				delete it or keep it?
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<div class="flex justify-center gap-2">
				<Button variant="destructive" onclick={confirmCleanup}>
					<Trash class="mr-2 h-4 w-4" />
					Delete
				</Button>
				<Button variant="outline" onclick={cancelCleanup}>
					<X class="mr-2 h-4 w-4" />
					Keep
				</Button>
			</div>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
