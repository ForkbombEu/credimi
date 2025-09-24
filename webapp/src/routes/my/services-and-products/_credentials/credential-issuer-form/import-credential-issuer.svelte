<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { AlertCircle, CheckCircle2, Download, Loader2 } from 'lucide-svelte';

	import type { CredentialIssuersResponse } from '@/pocketbase/types';

	import { Alert, AlertDescription } from '@/components/ui/alert';
	import { Button } from '@/components/ui/button';
	import { Card, CardContent } from '@/components/ui/card';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	//

	type Props = {
		organizationId: string;
		currentIssuers?: CredentialIssuersResponse[];
		after?: Snippet<[{ formState: typeof formState }]>;
		onImported?: (issuerId: string) => void;
	};

	let { organizationId, currentIssuers = [], after, onImported }: Props = $props();

	//

	//

	let isProcessingImport = $state(false);
	let importUrl = $state('');
	let importError = $state('');
	let importSuccess = $state('');
	let showCredentialsList = $state(false);
	let credentialIssuerId = $state<string | undefined>();
	let importMode = $state<boolean | null>(null); // null = no choice made, true = import, false = manual
	let existingIssuer = $state<CredentialIssuersResponse | undefined>();

	function useExistingIssuer() {
		if (existingIssuer && onImported) {
			onImported(existingIssuer.id);
			// Set up state as if we imported it
			credentialIssuer = existingIssuer;
			credentialIssuerId = existingIssuer.id;
			importSuccess = `Using existing credential issuer: ${existingIssuer.name || 'credential issuer'} from ${existingIssuer.url}.`;
			importError = '';
			importUrl = '';
			importMode = true;
			existingIssuer = undefined;
		}
	}

	function resetImport() {
		credentialIssuer = undefined;
		credentialIssuerId = undefined;
		importUrl = '';
		importError = '';
		importSuccess = '';
		showCredentialsList = false;
		importMode = null;
		existingIssuer = undefined;
	}

	function getCredentials(issuer: CredentialIssuersResponse) {
		try {
			// Access the expanded credentials safely
			const expand = issuer.expand;
			if (
				expand &&
				typeof expand === 'object' &&
				'credentials_via_credential_issuer' in expand
			) {
				return (expand as { credentials_via_credential_issuer: unknown[] })
					.credentials_via_credential_issuer;
			}
			return [];
		} catch {
			return [];
		}
	}

	function getCredentialName(credential: unknown): string {
		if (credential && typeof credential === 'object' && credential !== null) {
			const cred = credential as { display_name?: string; name?: string };
			return cred.display_name || cred.name || 'Unknown credential';
		}
		return 'Unknown credential';
	}

	function getCredentialFormat(credential: unknown): string | null {
		if (credential && typeof credential === 'object' && credential !== null) {
			const cred = credential as { format?: string };
			return cred.format || null;
		}
		return null;
	}

	async function importFromUrl() {
		if (!importUrl.trim()) return;

		isProcessingImport = true;
		importError = '';
		importSuccess = '';

		try {
			// Check if URL already exists for this organization
			const existingIssuers = await pb.collection('credential_issuers').getFullList({
				filter: `owner = "${organizationId}" && url = "${importUrl.trim()}"`,
				expand: 'credentials_via_credential_issuer'
			});

			if (existingIssuers.length > 0) {
				existingIssuer = existingIssuers[0];
				importError = `A credential issuer with this URL already exists in your organization: "${existingIssuer.name || existingIssuer.url}". You can edit the existing one instead of creating a new one.`;
				return;
			}

			await pb.send('/api/credentials_issuers/start-check', {
				method: 'POST',
				body: {
					credentialIssuerUrl: importUrl.trim()
				}
			});

			credentialIssuer = await getCreatedCredentialIssuer(importUrl.trim());
			credentialIssuerId = credentialIssuer.id;

			// Notify parent component about the imported issuer
			if (onImported && credentialIssuer.id) {
				onImported(credentialIssuer.id);
			}

			// Import is successful regardless of credentials found
			importSuccess = `Successfully created new credential issuer: ${credentialIssuer.name || 'credential issuer'} from ${credentialIssuer.url}.`;
			importUrl = '';
			importMode = true;

			// eslint-disable-next-line @typescript-eslint/no-explicit-any
		} catch (error: any) {
			if (error?.response?.error?.code === 404) {
				importError = m.Could_not_import_credential_issuer_well_known();
			} else {
				importError =
					error?.response?.error?.message || 'Failed to import credential issuer';
			}
		} finally {
			isProcessingImport = false;
		}
	}

	async function getCreatedCredentialIssuer(url: string) {
		const record = await pb.collection('credential_issuers').getFullList({
			filter: `owner = "${organizationId}" && url = "${url}"`,
			expand: 'credentials_via_credential_issuer'
		});
		if (record.length != 1) throw new Error('Unexpected number of records');
		return record[0];
	}

	//

	let credentialIssuer = $state<CredentialIssuersResponse | undefined>();

	// Get the latest version of the credential issuer from parent's data
	const currentCredentialIssuer = $derived(
		credentialIssuerId
			? currentIssuers.find((issuer) => issuer.id === credentialIssuerId)
			: undefined
	);

	// Use the most up-to-date issuer data (from parent subscription or local)
	const displayIssuer = $derived(currentCredentialIssuer || credentialIssuer);

	// Success message that updates reactively
	const currentImportSuccess = $derived.by(() => {
		if (!displayIssuer || !importSuccess) return importSuccess;

		const credentials = getCredentials(displayIssuer);
		const baseMessage = `Successfully imported credential issuer from ${displayIssuer.url}.`;

		if (credentials.length > 0) {
			return `${baseMessage} Found ${credentials.length} credential${credentials.length === 1 ? '' : 's'}.`;
		} else {
			return `${baseMessage} Discovering credentials...`;
		}
	});

	// Check if we're still discovering credentials
	const isDiscovering = $derived(
		displayIssuer && getCredentials(displayIssuer).length === 0 && !!importSuccess
	);

	const formState = $derived({
		loading: isProcessingImport,
		credentialIssuer: displayIssuer,
		importMode,
		showManualForm: importMode === false
	});
</script>

<Card class="bg-secondary border-purple-outline/20 mb-8">
	<CardContent class="space-y-4 p-6">
		<div class="flex items-start gap-3">
			<Download class="text-secondary-foreground mt-0.5 h-5 w-5 flex-shrink-0" />
			<div class="flex-1 space-y-1">
				<h4 class="text-secondary-foreground text-base font-medium">
					<strong>Optional</strong>: Import new credential issuer
				</h4>
				<p class="text-secondary-foreground/80 text-sm">
					Import a new credential issuer by providing its URL. This will create a new
					issuer record, fetch the issuer's well-known metadata and automatically discover
					available credentials. If the URL already exists in your organization, you'll be
					given the option to use the existing one.
				</p>
			</div>
		</div>

		{#if importError}
			<Alert variant="destructive">
				<AlertCircle class="h-4 w-4" />
				<AlertDescription>
					{importError}
					{#if existingIssuer}
						<div class="mt-2 text-sm">
							<strong>Existing issuer details:</strong>
							<br />
							<span class="font-medium">Name:</span>
							{existingIssuer.name || 'Unnamed'}
							<br />
							<span class="font-medium">URL:</span>
							{existingIssuer.url}
							<br />
							<span class="font-medium">Created:</span>
							{new Date(existingIssuer.created).toLocaleDateString()}
						</div>
						<div class="mt-3">
							<Button
								type="button"
								variant="outline"
								size="sm"
								onclick={useExistingIssuer}
							>
								Use Existing Issuer
							</Button>
						</div>
					{/if}
				</AlertDescription>
			</Alert>
		{/if}

		{#if importSuccess}
			<Alert class="border-green-200 bg-green-50">
				<CheckCircle2 class="h-4 w-4 text-green-600" />
				<AlertDescription class="text-green-800">
					{currentImportSuccess}
					{#if displayIssuer}
						{@const credentials = getCredentials(displayIssuer)}
						{#if credentials.length > 0}
							<button
								class="ml-2 text-green-700 underline underline-offset-2 hover:no-underline"
								onclick={() => (showCredentialsList = !showCredentialsList)}
								type="button"
							>
								{showCredentialsList ? 'Hide' : 'Show'} credentials
							</button>
						{/if}
					{/if}
					{#if isDiscovering}
						<span class="ml-2 text-xs text-green-600">(discovering more...)</span>
					{/if}
				</AlertDescription>
			</Alert>

			{#if showCredentialsList && displayIssuer}
				{@const credentials = getCredentials(displayIssuer)}
				{#if credentials.length > 0}
					<div class="rounded-md border border-green-200 bg-green-50 p-3">
						<ul class="space-y-1">
							{#each credentials as credential}
								<li class="text-sm text-green-800">
									• {getCredentialName(credential)}
									{#if getCredentialFormat(credential)}
										<span class="ml-1 text-xs text-green-600">
											({getCredentialFormat(credential)})
										</span>
									{/if}
								</li>
							{/each}
						</ul>
					</div>
				{/if}
			{/if}
		{/if}

		{#if !displayIssuer}
			<div class="flex gap-2">
				<div class="flex-1">
					<input
						type="url"
						bind:value={importUrl}
						placeholder="Enter credential issuer URL (e.g., https://example.com/issuer)"
						class="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-10 w-full rounded-md border px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
					/>
				</div>
				<Button
					type="button"
					variant="outline"
					onclick={importFromUrl}
					disabled={isProcessingImport || !importUrl.trim()}
				>
					{#if isProcessingImport}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
						Importing...
					{:else}
						<Download class="mr-2 h-4 w-4" />
						{m.Import()}
					{/if}
				</Button>
			</div>
		{:else}
			<div class="flex justify-center">
				<Button variant="outline" onclick={resetImport}>Import another issuer</Button>
			</div>
		{/if}
	</CardContent>
</Card>

{@render after?.({ formState })}
