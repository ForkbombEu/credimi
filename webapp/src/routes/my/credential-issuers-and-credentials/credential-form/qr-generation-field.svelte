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

	import QrGenerationField from '@/components/qr-generation-field.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import * as Tabs from '@/components/ui/tabs';
	import Field from '@/forms/fields/field.svelte';
	import { m } from '@/i18n';
	import { QrCode } from '@/qr';

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
			label:
				m.Static() +
				(credential?.deeplink || isDeeplinkTainted ? '' : ` (${m.Imported()})`),
			value: 'static'
		},
		dynamic: { label: m.Dynamic(), value: 'dynamic' }
	} satisfies Record<FieldMode, { label: string; value: FieldMode }>);

	/* Default */

	onMount(() => {
		if (credential?.name && !credential.deeplink) {
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
		} else if (credential?.name) {
			activeTab = 'static';
		} else {
			activeTab = 'dynamic';
		}
	});
</script>

<Tabs.Root bind:value={activeTab} class="w-full">
	<Tabs.List class="mb-4 w-full gap-1 bg-black/10">
		{#each Object.values(modesTabs) as tab (tab)}
			<Tabs.Trigger
				class={[
					'grow basis-1',
					'data-[state=inactive]:bg-white/40 data-[state=inactive]:text-black data-[state=inactive]:hover:bg-white/80'
				]}
				value={tab.value}
			>
				{tab.label}
			</Tabs.Trigger>
		{/each}
	</Tabs.List>

	<Tabs.Content value={modesTabs.static.value}>
		{@render tabDescription(m.Configure_static_deeplink())}

		<div class="flex gap-4">
			<div class="grow">
				<Field
					{form}
					name="deeplink"
					options={{
						placeholder: m.openid_credential_offer_placeholder()
					}}
				/>
			</div>
			<div class="pt-[22px]">
				<QrCode src={deeplinkState.current} placeholder={m.Type_to_generate_QR_code()} />
			</div>
		</div>
	</Tabs.Content>

	<Tabs.Content value="dynamic" class="min-w-0">
		{@render tabDescription(m.Configure_dynamic_credential_offer())}

		<QrGenerationField
			{form}
			fieldName="yaml"
			label={m.YAML_Configuration()}
			description={m.Provide_credential_configuration_in_YAML_format()}
			placeholder={m.Run_the_code_to_generate_QR_code()}
			successMessage={m.Compliance_Test_Completed_Successfully()}
			loadingMessage={m.Running_compliance_test()}
		/>
	</Tabs.Content>
</Tabs.Root>

{#snippet tabDescription(label: string)}
	<div class="-mx-4 -mt-3 mb-4 border-b border-b-blue-200 px-4 pb-2">
		<T class="text-center text-sm text-blue-500">{label}</T>
	</div>
{/snippet}
