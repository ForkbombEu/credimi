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
	};

	let { organizationId }: Props = $props();

	//

	let isImporting = $state(false);
	let importedIssuer = $state<CredentialIssuersResponse>();
	let showCleanupDialog = $state(false);

	async function discard() {
		if (!importedIssuer) return;
		await pb.collection('credential_issuers').delete(importedIssuer.id);
		importedIssuer = undefined;
		showCleanupDialog = false;
	}

	function cancelDiscard() {
		importedIssuer = undefined;
		showCleanupDialog = false;
	}
</script>

<Sheet
	beforeClose={(prevent) => {
		if (!importedIssuer) return;
		prevent();
		showCleanupDialog = true;
	}}
>
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
				onImport={(issuer) => {
					importedIssuer = issuer;
				}}
				bind:isLoading={isImporting}
			/>

			{#if isImporting}
				<div
					class={[
						'rounded-lg border bg-slate-200 py-10 text-center',
						'flex items-center justify-center gap-2',
						'animate-pulse'
					]}
				>
					<Spinner size={16} />
					<T class="text-muted-foreground">
						{m.importing()}
					</T>
				</div>
			{:else if importedIssuer}
				<div class="space-y-4">
					<p class="text-muted-foreground text-sm">
						Review and edit the imported credential issuer details below:
					</p>

					<CollectionForm
						collection="credential_issuers"
						recordId={importedIssuer.id}
						initialData={importedIssuer}
						fieldsOptions={{
							exclude: [
								'published',
								'owner',
								'url',
								'canonified_name',
								'imported',
								'workflow_url'
							]
						}}
						onSuccess={() => {
							importedIssuer = undefined;
							closeSheet();
						}}
						uiOptions={{
							showToastOnSuccess: true
						}}
					/>
				</div>
			{:else}
				<!-- Manual form -->
				<CollectionForm
					collection="credential_issuers"
					fieldsOptions={{
						hide: {
							owner: organizationId
						},
						exclude: ['published', 'imported', 'canonified_name', 'workflow_url']
					}}
					onSuccess={closeSheet}
					uiOptions={{
						showToastOnSuccess: true
					}}
				/>
			{/if}
		</div>

		<!-- Cleanup Confirmation Dialog -->
		<AlertDialog.Root bind:open={showCleanupDialog}>
			<AlertDialog.Content>
				<AlertDialog.Header>
					<AlertDialog.Title>Delete imported credential issuer?</AlertDialog.Title>
					<AlertDialog.Description>
						You have imported a credential issuer but haven't saved it yet. Would you
						like to delete it or keep it?
					</AlertDialog.Description>
				</AlertDialog.Header>
				<AlertDialog.Footer>
					<div class="flex justify-center gap-2">
						<Button
							variant="destructive"
							onclick={async () => {
								await discard();
								closeSheet();
							}}
						>
							<Trash class="mr-2 h-4 w-4" />
							Delete
						</Button>
						<Button
							variant="outline"
							onclick={() => {
								cancelDiscard();
								closeSheet();
							}}
						>
							<X class="mr-2 h-4 w-4" />
							Keep
						</Button>
					</div>
				</AlertDialog.Footer>
			</AlertDialog.Content>
		</AlertDialog.Root>
	{/snippet}
</Sheet>
