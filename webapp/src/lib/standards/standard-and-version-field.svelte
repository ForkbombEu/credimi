<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T extends Record<string, unknown>">
	import { SelectField } from '@/forms/fields';
	import { m } from '@/i18n';
	import type { FormPath, SuperForm } from 'sveltekit-superforms';
	import { getStandardsAndTestSuites } from '.';
	import { Either } from 'effect';

	//

	type Props = {
		form: SuperForm<T>;
		name: FormPath<T>;
	};

	const { form, name }: Props = $props();
	const formData = $derived(form.form);

	//

	type Option = {
		value: string;
		label: string;
	};

	let options: Option[] = $state([]);

	getStandardsAndTestSuites().then((response) => {
		if (!Either.isRight(response)) return;
		const standards = response.right;
		options = standards.flatMap((standard) =>
			standard.versions.map((version) => ({
				value: `${standard.uid}/${version.uid}`,
				label: `${standard.name} – ${version.name}`
			}))
		);
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
