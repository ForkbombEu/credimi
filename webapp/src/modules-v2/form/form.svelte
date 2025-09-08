<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="T extends t.GenericRecord">
	import type { Snippet } from 'svelte';
	import type { HTMLFormAttributes } from 'svelte/elements';

	// eslint-disable-next-line @typescript-eslint/no-unused-vars
	import { form as _form, types as t } from '#';

	import Button from '@/components/ui-custom/button.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import { Alert } from '@/components/ui/alert';
	import { m } from '@/i18n';

	//

	type Props = {
		// eslint-disable-next-line no-undef
		form: _form.Instance<T>;
		children?: Snippet;
	};

	let { form, children }: Props = $props();

	const superform = form.attachSuperform();
	const { enhance } = superform;

	type Enctype = HTMLFormAttributes['enctype'];
	const enctype: Enctype = $derived(
		form.superform?.options.dataType == 'form' ? 'multipart/form-data' : undefined
	);
</script>

<form use:enhance method="post" {enctype}>
	{@render children?.()}

	{#if form.submitError}
		<Alert variant="destructive">
			{form.submitError.message}
		</Alert>
	{/if}

	<Button type="submit" disabled={!form.valid}>
		{#if !form.delayed}
			{m.Submit()}
		{:else}
			<Spinner size={16} />
			{m.Please_wait()}
		{/if}
	</Button>
</form>
