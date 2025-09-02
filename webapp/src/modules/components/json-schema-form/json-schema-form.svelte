<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { BasicForm } from '@sjsf/form';
	import { setThemeContext } from '@sjsf/shadcn-theme';
	import * as components from '@sjsf/shadcn-theme/default';

	import type { JsonSchemaForm } from './create-form.svelte';

	type Props = {
		form: JsonSchemaForm;
		hideSubmitButton?: boolean;
		class?: string;
	};

	const { form, hideSubmitButton, class: className }: Props = $props();

	setThemeContext({ components });
</script>

<BasicForm
	{form}
	class={[
		'space-y-6',
		className,
		{
			'[&>button[type=submit]]:hidden': hideSubmitButton
		}
	]}
/>
