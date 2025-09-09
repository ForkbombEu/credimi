<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { FormPathLeaves, SuperForm } from 'sveltekit-superforms';

	import type { FieldOptions } from '@/forms/fields/types';
	import type { CredentialIssuersResponse, CredentialsRecord } from '@/pocketbase/types';
	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	import type { GenericRecord } from '@/utils/types';

	import * as Form from '@/components/ui/form';
	import * as Tabs from '@/components/ui/tabs';
	import FieldWrapper from '@/forms/fields/parts/fieldWrapper.svelte';

	import DynamicTab from './dynamic-tab.svelte';
	import QrDisplay from './qr-display.svelte';
	import StaticTab from './static-tab.svelte';

	//

	interface Props {
		form: SuperForm<Data>;
		deeplinkName: FormPathLeaves<Data, string>;
		yaml: FormPathLeaves<Data, string>;
		credential: CredentialsRecord;
		credentialIssuer: CredentialIssuersResponse;
		options?: Partial<FieldOptions>;
	}

	const {
		form,
		deeplinkName,
		yaml,
		options = {},
		credential,
		credentialIssuer
	}: Props = $props();

	const { form: formData } = $derived(form);

	let activeTab = $state('static');
	let hasInitialized = $state(false);

	const shouldAutoSwitchToDynamic = $derived.by(() => {
		const yamlValue = (($formData as Record<string, unknown>)[yaml] as string) || '';
		const deeplinkValue = (($formData as Record<string, unknown>)['deeplink'] as string) || '';
		return !hasInitialized && !deeplinkValue && !!yamlValue && yamlValue.length > 0;
	});

	$effect(() => {
		if (shouldAutoSwitchToDynamic && activeTab === 'static') {
			activeTab = 'dynamic';
		}
		hasInitialized = true;
	});

	let isSubmittingCompliance = $state(false);
	let credentialOffer = $state<string | null>(null);
	let workflowSteps = $state<unknown[] | null>(null);
	let workflowOutput = $state<unknown[] | null>(null);
	let workflowError = $state<string | null>(null);

	let staticDeepLink = $state<string>('');
	let staticShowUrl = $state<boolean>(true);

	const qrStatus = $derived(() => {
		const yamlValue = (($formData as Record<string, unknown>)[yaml] as string) || '';
		const deeplinkValue =
			(($formData as Record<string, unknown>)[deeplinkName] as string) || '';

		if (deeplinkValue) {
			return {
				message: 'Using deeplink',
				variant: 'success' as const
			};
		}

		if (yamlValue) {
			if (credentialOffer) {
				return {
					message: 'Using YAML',
					variant: 'success' as const
				};
			}
			if (workflowError) {
				return {
					message: 'Correct YAML',
					variant: 'destructive' as const
				};
			}
			return {
				message: 'Configure YAML',
				variant: 'default' as const
			};
		}
		return {
			message: 'Using default',
			variant: 'default' as const
		};
	});

	const staticTabLabel = $derived(() => {
		const baseLabel = 'Static Generation';
		if (activeTab === 'static') {
			return staticShowUrl === false ? `${baseLabel} (Custom)` : `${baseLabel} (Default)`;
		}
		return baseLabel;
	});

	const dynamicTabLabel = $derived(() => {
		const baseLabel = 'Dynamic Generation';
		if (activeTab === 'dynamic') {
			return credentialOffer ? `${baseLabel} (Active)` : baseLabel;
		}
		return baseLabel;
	});
</script>

<Form.Field {form} name={deeplinkName}>
	<FieldWrapper field={deeplinkName} {options}>
		{#snippet children()}
			{#if activeTab === 'static' && staticDeepLink}
				<div class="mb-6">
					<div class="mb-3 flex items-center gap-2 text-sm">
						<div
							class="h-2 w-2 rounded-full {qrStatus().variant === 'success'
								? 'bg-green-500'
								: qrStatus().variant === 'destructive'
									? 'bg-red-500'
									: 'bg-blue-500'}"
						></div>
						<span class="text-muted-foreground">{qrStatus().message}</span>
					</div>
					<QrDisplay src={staticDeepLink} showUrl={staticShowUrl} />
				</div>
			{:else if activeTab === 'dynamic' && credentialOffer}
				<div class="mb-6">
					<div class="mb-3 flex items-center gap-2 text-sm">
						<div
							class="h-2 w-2 rounded-full {qrStatus().variant === 'success'
								? 'bg-green-500'
								: qrStatus().variant === 'destructive'
									? 'bg-red-500'
									: 'bg-blue-500'}"
						></div>
						<span class="text-muted-foreground">{qrStatus().message}</span>
					</div>
					<QrDisplay src={credentialOffer} />
				</div>
			{:else}
				<div class="mb-6">
					<div class="mb-3 flex items-center gap-2 text-sm">
						<div
							class="h-2 w-2 rounded-full {qrStatus().variant === 'success'
								? 'bg-green-500'
								: qrStatus().variant === 'destructive'
									? 'bg-red-500'
									: 'bg-blue-500'}"
						></div>
						<span class="text-muted-foreground">{qrStatus().message}</span>
					</div>
					<div
						class="border-muted-foreground/25 bg-muted/10 flex h-60 w-60 items-center justify-center rounded-md border border-dashed"
					>
						<span class="text-muted-foreground text-sm">QR code will appear here</span>
					</div>
				</div>
			{/if}

			<Tabs.Root bind:value={activeTab} class="w-full">
				<Tabs.List class="grid w-full grid-cols-2">
					<Tabs.Trigger value="static">{staticTabLabel()}</Tabs.Trigger>
					<Tabs.Trigger value="dynamic">{dynamicTabLabel()}</Tabs.Trigger>
				</Tabs.List>

				<Tabs.Content value="static" class="mt-4">
					<div class="mb-3 text-sm">
						{#if staticShowUrl === false}
							You've configured a custom deeplink. Clear the field below to use the
							default configuration.
						{:else}
							Pre-configured credential offer with fixed parameters. Enter a custom
							deeplink below to override.
						{/if}
					</div>
					<StaticTab
						{form}
						name={deeplinkName}
						{credential}
						{credentialIssuer}
						{options}
						onDeepLinkChange={(deepLink, showUrl) => {
							staticDeepLink = deepLink;
							staticShowUrl = showUrl;
						}}
					/>
				</Tabs.Content>

				<Tabs.Content value="dynamic" class="mt-4 min-w-0">
					<div class="text-muted-foreground mb-3 text-sm">
						{#if credentialOffer}
							QR code generated successfully from your YAML configuration.
						{:else}
							Configure the YAML below to generate a dynamic credential offer with
							runtime parameters.
						{/if}
					</div>
					<div class="min-w-0">
						<DynamicTab
							{form}
							name={yaml}
							bind:isSubmittingCompliance
							bind:credentialOffer
							bind:workflowSteps
							bind:workflowOutput
							bind:workflowError
						/>
					</div>
				</Tabs.Content>
			</Tabs.Root>
		{/snippet}
	</FieldWrapper>
</Form.Field>
