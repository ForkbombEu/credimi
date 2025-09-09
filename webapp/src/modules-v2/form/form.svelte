<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T extends t.GenericRecord">
	import type { Snippet } from 'svelte';

	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	import { form as _form, types as t } from '#';

	import { Form } from '@/forms';

	//

	type Props = {
		// eslint-disable-next-line no-undef
		form: _form.Instance<T>;
		children?: Snippet;
		hide?: ('error' | 'submit_button')[];
	};

	let { form, children, hide = [] }: Props = $props();

	const superform = form.attachSuperform();
</script>

<Form form={superform} {hide}>
	{@render children?.()}
</Form>
