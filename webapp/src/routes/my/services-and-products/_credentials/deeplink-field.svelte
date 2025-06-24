<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { GenericRecord } from '@/utils/types';
	import * as Form from '@/components/ui/form';
	import type { FormPathLeaves, SuperForm } from 'sveltekit-superforms';
	import { stringProxy } from 'sveltekit-superforms';
	import FieldWrapper from '@/forms/fields/parts/fieldWrapper.svelte';
	import type { FieldOptions } from '@/forms/fields/types';
	import { Input } from '@/components/ui/input';
	import { QrCode } from '@/qr';
	import { String } from 'effect';
	import type { CredentialIssuersResponse, CredentialsRecord } from '@/pocketbase/types';
	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, string>;
		credential: CredentialsRecord;
		credentialIssuer: CredentialIssuersResponse;
		options?: Partial<FieldOptions>;
	}

	const { form, name, options = {}, credential, credentialIssuer }: Props = $props();

	const { form: formData } = $derived(form);
	const valueProxy = $derived(stringProxy(formData, name, { empty: 'undefined' }));
	let qrLink = $state('');
	let deepLink = $state('');

	$effect(() => {
		if (formData) {
			const unsubscribe = formData.subscribe((d) => {
				if (
					'deeplink' in d &&
					typeof d.deeplink === 'string' &&
					String.isNonEmpty(d.deeplink)
				) {
					deepLink = qrLink = d.deeplink;
				} else {
					qrLink = createIntentUrl(credentialIssuer?.url, credential.type!);
					deepLink = '';
				}
			});
			return unsubscribe;
		}
	});

	//

	function createIntentUrl(issuer: string | undefined, type: string): string {
		const data = {
			credential_configuration_ids: [type],
			credential_issuer: issuer
		};
		const credentialOffer = encodeURIComponent(JSON.stringify(data));
		return `openid-credential-offer://?credential_offer=${credentialOffer}`;
	}
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#snippet children()}
			<div class="flex items-stretch">
				<QrCode src={qrLink} cellSize={10} class={['w-60 rounded-md']} />
				{#if qrLink !== deepLink}
					<div class="w-60 break-all pt-4 text-xs">
						<a href={qrLink} target="_self">{qrLink}</a>
					</div>
				{/if}
			</div>
			<Input {...options as GenericRecord} bind:value={$valueProxy} />
		{/snippet}
	</FieldWrapper>
</Form.Field>
