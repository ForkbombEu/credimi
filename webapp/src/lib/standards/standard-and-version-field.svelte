<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T extends Record<string, unknown>">
	import { SelectField } from '@/forms/fields';
	import { m } from '@/i18n';
	import type { FormPath, SuperForm } from 'sveltekit-superforms';
	import { getStandardsAndVersionsFlatOptionsList } from '.';
	import type { SelectOption } from '@/components/ui-custom/utils';

	//

	type Props = {
		form: SuperForm<T>;
		name: FormPath<T>;
	};

	const { form, name }: Props = $props();
	const formData = $derived(form.form);

	//

	let options: SelectOption<string>[] = $state([]);

	getStandardsAndVersionsFlatOptionsList().then((o) => {
		options = o;
	});
</script>

<!-- `any` is needed to avoid `type instantiation is excessively deep and possibly infinite` -->
<SelectField
	{form}
	name={name as any}
	options={{
		items: options,
		label: m.Compliance_standard(),
		placeholder: m.Select_a_standard_and_version(),
		description: `${m.eg()}: OpenID4VP Verifier - Draft 23`
	}}
/>
