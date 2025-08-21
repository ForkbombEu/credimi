<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { m } from '@/i18n';
	import { Button } from '@/components/ui/button';
	import { Input } from '@/components/ui/input';
	import { Label } from '@/components/ui/label';
	import { CredimiClient, CredimiClientError } from '$lib/credimiClient';
	import type { GenerateApiKeyResponse } from '$lib/credimiClient';
	import { pb, currentUser } from '@/pocketbase';
	import { CollectionManager, RecordDelete } from '@/collections-components/manager';
	import type { ApiKeysResponse } from '@/pocketbase/types';
	import * as AlertDialog from '@/components/ui/alert-dialog';
	import * as Table from '@/components/ui/table';
	import * as Card from '@/components/ui/card';
	import CopyButton from '@/components/ui-custom/copyButton.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { Separator } from '@/components/ui/separator';
	import { AlertTriangle, Trash2 } from 'lucide-svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import { z } from 'zod';

	const apiKeyNameSchema = z.object({
		name: z.string()
			.trim()
			.min(1, m.Please_enter_an_API_key_name())
			.max(30, m.API_key_name_must_not_be_more_than_30_characters())
	});

	const credimiClient = new CredimiClient(pb);

	let apiKeyName = $state('');
	let isLoading = $state(false);
	let apiKeyDialogOpen = $state(false);
	let generatedApiKey = $state<GenerateApiKeyResponse | null>(null);
	let generatedApiKeyName = $state<string>('');
	let error = $state<string | null>(null);
	let dialogTimer = $state(10);
	let timerInterval: ReturnType<typeof setInterval> | null = null;

	async function generateApiKey() {
		// Validate using Zod schema
		const validation = apiKeyNameSchema.safeParse({ name: apiKeyName });
		
		if (!validation.success) {
			error = validation.error.errors[0].message;
			return;
		}

		isLoading = true;
		error = null;
		generatedApiKey = null;

		try {
			const result = await credimiClient.generateApiKey({ name: validation.data.name });
			generatedApiKey = result;
			generatedApiKeyName = validation.data.name; // Use validated and trimmed name
			apiKeyName = ''; // Clear the form

			// Open dialog and start timer
			apiKeyDialogOpen = true;
			dialogTimer = 5;
			startTimer();
		} catch (err) {
			if (err instanceof CredimiClientError) {
				error = `${m.Error()}: ${err.data.Message}`;
			} else {
				error = m.An_unexpected_error_occurred();
			}
		} finally {
			isLoading = false;
		}
	}

	function startTimer() {
		timerInterval = setInterval(() => {
			dialogTimer--;
			if (dialogTimer <= 0) {
				clearInterval(timerInterval!);
				timerInterval = null;
			}
		}, 1000);
	}

	function closeDialog() {
		if (dialogTimer <= 0) {
			apiKeyDialogOpen = false;
			if (timerInterval) {
				clearInterval(timerInterval);
				timerInterval = null;
			}
		}
	}

	// Cleanup timer on destroy
	$effect(() => {
		return () => {
			if (timerInterval) {
				clearInterval(timerInterval);
			}
		};
	});
</script>

<div class="space-y-6">
	<div>
		<T tag="h4">{m.API_Keys()}</T>
		<T class="text-muted-foreground">
			{m.API_keys_for_automated_testing()}
		</T>
	</div>

	<Separator />

	<!-- API Key Creation Section -->
	<div>
		<T tag="h4">{m.Create_New_API_Key()}</T>
		<T class="text-muted-foreground"
			>{m.Generate_a_new_API_key_for_accessing_the_services()}</T
		>
	</div>
	<div class="flex flex-col gap-4">
		<Label for="apiKeyName">{m.API_Key_Name()}</Label>
		<Input
			id="apiKeyName"
			bind:value={apiKeyName}
			placeholder={m.Enter_a_descriptive_name_for_your_API_key()}
			disabled={isLoading}
			maxlength={30}
		/>
		<Button class="w-fit" onclick={generateApiKey} disabled={isLoading || !apiKeyNameSchema.safeParse({ name: apiKeyName }).success}>
			{isLoading ? m.Generating() : m.Generate_API_Key()}
		</Button>
	</div>

	{#if error}
		<div class="flex items-center gap-2 rounded-md border border-red-200 bg-red-50 p-3">
			<Icon src={AlertTriangle} class="text-red-600" size="sm" />
			<T class="text-sm text-red-700">{error}</T>
		</div>
	{/if}

	<!-- Existing API Keys Table -->

	<Separator />
	<div>
		<T tag="h4">{m.Your_API_Keys()}</T>
		<T class="text-muted-foreground"
			>{m.Here_are_your_active_API_keys()}</T
		>
	</div>
	<Card.Content>
		{#if $currentUser?.id}
			<CollectionManager
				collection="api_keys"
				queryOptions={{ filter: `user = "${$currentUser.id}"` }}
			>
				{#snippet records({ records })}
					{#if records.length === 0}
						<div class="py-8 text-center">
							<T class="text-muted-foreground">{m.No_API_keys_found()}</T>
							<T class="text-muted-foreground text-sm"
								>{m.Create_your_first_API_key_above()}</T
							>
						</div>
					{:else}
						<Table.Root>
							<Table.Header>
								<Table.Row>
									<Table.Head>#</Table.Head>
									<Table.Head>{m.Name()}</Table.Head>
									<Table.Head>{m.Created()}</Table.Head>
									<Table.Head class="w-[100px]">{m.Actions()}</Table.Head>
								</Table.Row>
							</Table.Header>
							<Table.Body>
								{#each records as apiKey, index (apiKey.id)}
									<Table.Row>
										<Table.Cell class="font-medium">{index + 1}</Table.Cell>
										<Table.Cell>{apiKey.name}</Table.Cell>
										<Table.Cell class="text-muted-foreground text-sm">
											{new Date(apiKey.created).toLocaleDateString()}
										</Table.Cell>
										<Table.Cell>
											<RecordDelete record={apiKey}>
												{#snippet button({
													triggerAttributes,
													icon: DeleteIcon
												})}
													<Button
														variant="outline"
														size="sm"
														class="h-8 w-8 p-0 text-red-600 hover:text-red-700"
														{...triggerAttributes}
													>
														<Icon src={Trash2} size="sm" />
													</Button>
												{/snippet}
											</RecordDelete>
										</Table.Cell>
									</Table.Row>
								{/each}
							</Table.Body>
						</Table.Root>
					{/if}
				{/snippet}
			</CollectionManager>
		{:else}
			<div class="py-8 text-center">
				<T class="text-muted-foreground">{m.Loading()}</T>
			</div>
		{/if}
	</Card.Content>
</div>

<!-- API Key Display Dialog -->
<AlertDialog.Root bind:open={apiKeyDialogOpen}>
	<AlertDialog.Content class="max-w-2xl">
		<AlertDialog.Header>
			<AlertDialog.Title>{m.Your_API_Key_Has_Been_Generated()}</AlertDialog.Title>
			<AlertDialog.Description>
				<div class="space-y-4">
					<div class="rounded-md border border-amber-200 bg-amber-50 p-4">
						<div class="flex items-start gap-2">
							<!-- <Icon src={AlertTriangle} class="mt-0.5 text-amber-600" size="sm" /> -->
							<div class="space-y-1">
								<T class="text-sm font-medium text-amber-800"
									>{m.Important_Security_Notice()}</T
								>
								<T class="text-sm text-amber-700">
									{m.This_API_key_will_only_be_shown_once()}
								</T>
							</div>
						</div>
					</div>

					{#if generatedApiKey}
						<div class="space-y-3">
							<div>
								<Label class="text-sm font-medium">{m.API_Key_Name()}</Label>
								<div class="mt-1 rounded border bg-gray-50 p-2 text-sm">
									{generatedApiKeyName || m.Unnamed_Key()}
								</div>
							</div>

							<div>
								<Label class="text-sm font-medium">{m.API_Key()}</Label>
								<div class="mt-1 rounded border bg-gray-50 p-3">
									<code class="break-all font-mono text-sm text-gray-800">
										{generatedApiKey.api_key}
									</code>
								</div>
								<div class="mt-2">
									<CopyButton textToCopy={generatedApiKey.api_key} size="sm">
										{m.Copy_API_Key()}
									</CopyButton>
								</div>
							</div>
						</div>
					{/if}
				</div>
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer class="flex items-center justify-between">
			<T class="text-muted-foreground text-sm">
				{#if dialogTimer > 0}
					{m.You_can_close_this_dialog_in_seconds({ dialogTimer })}
				{:else}
					{m.You_can_now_close_this_dialog()}
				{/if}
			</T>
			<Button disabled={dialogTimer > 0} onclick={closeDialog}>
				{dialogTimer > 0 ? m.Close_seconds({ dialogTimer }) : m.Close()}
			</Button>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
