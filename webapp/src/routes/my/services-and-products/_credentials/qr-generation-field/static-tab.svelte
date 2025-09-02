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

	import { Input } from '@/components/ui/input';

	//

	interface Props {
		form: SuperForm<Data>;
		name: FormPathLeaves<Data, string>;
		credential: CredentialsRecord;
		credentialIssuer: CredentialIssuersResponse;
		options?: Partial<FieldOptions>;
		onDeepLinkChange?: (deepLink: string, showUrl: boolean) => void;
	}

	const {
		form,
		name,
		options = {},
		credential,
		credentialIssuer,
		onDeepLinkChange
	}: Props = $props();
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

	$effect(() => {
		if (onDeepLinkChange) {
			const showUrl = String.isEmpty(fieldState.current);
			onDeepLinkChange(deepLink, showUrl);
		}
	});
</script>

<div class="space-y-4">
	<Input {...options} bind:value={fieldState.current} />
</div>
