<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	export type FieldMode = 'static' | 'dynamic';
</script>

<script lang="ts">
	import { createIntentUrl } from '$lib/credentials';
	import { onMount } from 'svelte';
	import { fromStore } from 'svelte/store';
	import { stringProxy, type SuperForm } from 'sveltekit-superforms';

	import type { CredentialIssuersResponse, CredentialsRecord } from '@/pocketbase/types';

	import T from '@/components/ui-custom/t.svelte';
	import * as Tabs from '@/components/ui/tabs';
	import Field from '@/forms/fields/field.svelte';
	import QrStateful from '@/qr/qr-stateful.svelte';

	import DynamicTab from './dynamic-tab.svelte';

	//

	interface Props {
		form: SuperForm<{ deeplink: string; yaml: string }>;
		credential?: CredentialsRecord;
		credentialIssuer: CredentialIssuersResponse;
		activeTab: FieldMode;
	}

	let { form, credential, credentialIssuer, activeTab = $bindable('static') }: Props = $props();

	/* Field value */

	const deeplinkState = fromStore(stringProxy(form, 'deeplink', { empty: 'undefined' }));
	const tainted = fromStore(form.tainted);
	const isDeeplinkTainted = $derived(Boolean(tainted.current?.deeplink));

	/* Tabs */

	const modesTabs = $derived({
		static: {
			label: 'Static' + (credential?.deeplink || isDeeplinkTainted ? '' : ' (Imported)'),
			value: 'static'
		},
		dynamic: { label: 'Dynamic', value: 'dynamic' }
	} satisfies Record<FieldMode, { label: string; value: FieldMode }>);

	/* Default */

	onMount(() => {
		if (credential?.key && !credential.deeplink) {
			form.form.update(
				(data) => {
					return { ...data, deeplink: createIntentUrl(credential, credentialIssuer.url) };
				},
				{ taint: false }
			);
		}
	});

	$effect(() => {
		if (credential?.yaml) {
			activeTab = 'dynamic';
		} else if (credential?.key) {
			activeTab = 'static';
		} else {
			activeTab = 'dynamic';
		}
	});
</script>

<Tabs.Root bind:value={activeTab} class="w-full">
	<Tabs.List class="mb-4 w-full">
		{#each Object.values(modesTabs) as tab}
			<Tabs.Trigger class="grow basis-1" value={tab.value}>
				{tab.label}
			</Tabs.Trigger>
		{/each}
	</Tabs.List>

	<Tabs.Content value={modesTabs.static.value}>
		<div class="flex gap-4">
			<div class="grow">
				<T class="mb-3">Configure a static deeplink with fixed parameters.</T>
				<Field
					{form}
					name="deeplink"
					options={{ placeholder: 'e.g. openid-credential-offer://?...' }}
				/>
			</div>
			<QrStateful src={deeplinkState.current} placeholder="Type to generate a QR code" />
		</div>
	</Tabs.Content>

	<Tabs.Content value="dynamic" class="min-w-0">
		<DynamicTab {form} />
	</Tabs.Content>
</Tabs.Root>
