<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { UnsubscribeFunc } from 'pocketbase';

	import { CheckCircle2, ChevronDownIcon, ChevronUpIcon, Download, Loader2 } from 'lucide-svelte';
	import { onDestroy, onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import { zod } from 'sveltekit-superforms/adapters';
	import { z } from 'zod';

	import type { CredentialIssuersResponse, CredentialsResponse } from '@/pocketbase/types';

	import { getCollectionManagerContext } from '@/collections-components/manager/collectionManagerContext';
	import Button from '@/components/ui-custom/button.svelte';
	import { Alert, AlertDescription } from '@/components/ui/alert';
	import { ScrollArea } from '@/components/ui/scroll-area/index';
	import { createForm, Form, SubmitButton } from '@/forms';
	import { Field } from '@/forms/fields';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';
	//

	type Props = {
		onImport?: (issuer: CredentialIssuersResponse) => void;
		isLoading?: boolean;
	};

	let { onImport, isLoading = $bindable(false) }: Props = $props();

	/* Credential issuer import */

	type Result = {
		credentialsNumber: number;
		record: CredentialIssuersResponse;
	};

	let result = $state<Result>();

	const form = createForm({
		adapter: zod(
			z.object({
				url: z
					.string()
					.regex(
						/^(https?:\/\/)?([\da-z.-]+)\.([a-z.]{2,6})([/\w .-]*)*\/?$/,
						'Please enter a valid URL'
					)
			})
		),
		onSubmit: async ({ form: { data } }) => {
			const res = await pb.send('/api/credentials_issuers/start-check', {
				method: 'POST',
				body: {
					credentialIssuerUrl: data.url.trim()
				}
			});
			result = res as Result;
			onImport?.(result.record);

			// Temp fix for imported issuer not showing
			updateManagerRecord(result.record);
		}
	});

	form.submitting.subscribe((value) => {
		isLoading = value;
	});

	/* Counting credentials */
	// We subscribe to all the credentials, then we filter them by the credential issuer

	const credentials = $state<CredentialsResponse[]>([]);

	let unsub: UnsubscribeFunc;
	onMount(async () => {
		unsub = await pb.collection('credentials').subscribe('*', (event) => {
			if (event.action === 'create') credentials.push(event.record);
		});
	});
	onDestroy(async () => {
		await unsub();
	});

	const validCredentials = $derived(
		credentials.filter((c) => c.credential_issuer === result?.record.id)
	);
	const hasAllCredentials = $derived(
		Boolean(result?.credentialsNumber && validCredentials.length == result.credentialsNumber)
	);
	$effect(() => {
		if (hasAllCredentials) unsub();
	});

	/* Credentials display */

	let showCredentials = $state(false);

	const credentialsText = $derived.by(() => {
		if (hasAllCredentials) {
			return m.All_credentials_imported_successfully() + ` (${validCredentials.length})`;
		} else {
			return m.Importing_credentials({
				count: validCredentials.length,
				total: result?.credentialsNumber ?? 0
			});
		}
	});

	/* Temp fix for imported issuer not showing */

	const { manager } = getCollectionManagerContext();

	function updateManagerRecord(record: CredentialIssuersResponse) {
		const records = manager.records as CredentialIssuersResponse[];
		const managerRecord = records.find((r) => r.id === result?.record.id);
		if (!managerRecord) return;
		const recordIndex = records.indexOf(managerRecord);
		// @ts-expect-error - manager.records is not typed
		manager.records[recordIndex] = record;
	}
</script>

<div class="bg-secondary border-purple-outline/20 mb-8 rounded-lg border">
	<div class="space-y-4 p-6">
		<div class="flex items-start gap-3">
			<Download class="text-secondary-foreground mt-0.5 h-5 w-5 shrink-0" />
			<div class="space-y-1">
				<h4 class="text-secondary-foreground text-base font-medium">
					<strong>{m.Optional()}</strong>: {m.Import_new_credential_issuer()}
				</h4>
				<p class="text-secondary-foreground/80 text-sm">
					{m.import_new_credential_issuer_description()}
				</p>
			</div>
		</div>

		{#if !result}
			<Form {form} hide={['submit_button', 'loading_state']} class="space-y-2">
				<div class="flex gap-2">
					<div class="grow">
						<Field
							{form}
							name="url"
							options={{
								placeholder:
									'Enter credential issuer URL (e.g., https://example.com/issuer)',
								hideLabel: true
							}}
						/>
					</div>
					<SubmitButton variant="outline" class="w-36 shrink-0">
						{#if isLoading}
							<Loader2 class="animate-spin" />
							{m.importing()}
						{:else}
							<Download />
							{m.Import()}
						{/if}
					</SubmitButton>
				</div>
			</Form>
		{:else}
			<Alert class="border-green-200 bg-green-50">
				<CheckCircle2 class="h-4 w-4 text-green-600" />
				<AlertDescription class="text-green-800">
					<p>{m.Credential_issuer_imported_successfully()}</p>
					{@render credentialsList(validCredentials, credentialsText)}
				</AlertDescription>
			</Alert>
		{/if}
	</div>
</div>

{#snippet credentialsList(credentials: CredentialsResponse[], text: string)}
	<div>
		<div class="flex items-center justify-between gap-2">
			<p>{text}</p>
			<Button
				variant="ghost"
				class="h-6 px-2"
				size="sm"
				onclick={() => (showCredentials = !showCredentials)}
			>
				{showCredentials ? m.hide() : m.show()}
				{#if showCredentials}
					<ChevronUpIcon size={14} />
				{:else}
					<ChevronDownIcon size={14} />
				{/if}
			</Button>
		</div>

		{#if showCredentials}
			<div class="mt-2" transition:slide>
				<ScrollArea class="mt h-[200px] rounded-md border px-3 py-2">
					<ul class="list-inside list-disc">
						{#each credentials as credential}
							<li>
								{credential.display_name || credential.name}
							</li>
						{/each}
					</ul>
				</ScrollArea>
			</div>
		{/if}
	</div>
{/snippet}
