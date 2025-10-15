<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { FormPathLeaves, SuperForm } from 'sveltekit-superforms';

	import { createIntentUrl } from '$lib/credentials';
	import { String } from 'effect';
	import { fromStore } from 'svelte/store';
	import { stringProxy } from 'sveltekit-superforms';

	import type { FieldOptions } from '@/forms/fields/types';
	import type { CredentialIssuersResponse, CredentialsRecord } from '@/pocketbase/types';
	import type { GenericRecord } from '@/utils/types';

	import * as Form from '@/components/ui/form';
	import { Input } from '@/components/ui/input';
	import FieldWrapper from '@/forms/fields/parts/fieldWrapper.svelte';
	import { QrCode } from '@/qr';

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

	const fieldProxy = $derived(stringProxy(formData, name, { empty: 'undefined' }));
	const fieldState = $derived(fromStore(fieldProxy));

	const deepLink = $derived.by(() => {
		if (String.isNonEmpty(fieldState.current)) {
			return fieldState.current;
		} else {
			return createIntentUrl(credential, credentialIssuer.url);
		}
	});
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#snippet children()}
			<Input {...options as GenericRecord} bind:value={fieldState.current} />
			<div class="flex flex-col items-stretch gap-4 pt-3 md:flex-row">
				<QrCode src={deepLink} cellSize={10} class="size-60 rounded-md border" />
				{#if String.isEmpty(fieldState.current)}
					<div class="max-w-60 break-all text-xs">
						<a href={deepLink} target="_self">{deepLink}</a>
					</div>
				{/if}
			</div>
		{/snippet}
	</FieldWrapper>
</Form.Field>
