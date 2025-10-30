<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SuperForm } from 'sveltekit-superforms';

	import QrGenerationField from '@/components/qr-generation-field.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	//

	interface Props {
		form: SuperForm<{ deeplink: string; yaml: string }>;
	}

	let { form }: Props = $props();
</script>

<div class="mb-4">
	<T>{m.Configure_dynamic_credential_offer()}</T>
</div>

<QrGenerationField
	{form}
	fieldName="yaml"
	label={m.YAML_Configuration()}
	description={m.Provide_credential_configuration_in_YAML_format()}
	placeholder={m.Run_the_code_to_generate_QR_code()}
	successMessage={m.Compliance_Test_Completed_Successfully()}
	loadingMessage={m.Running_compliance_test()}
	enableStructuredErrors={true}
/>
