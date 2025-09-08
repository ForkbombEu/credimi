<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { SuperForm } from 'sveltekit-superforms/client';

	import { types as t } from '#';
	import * as utils from '#/utils';

	const { getContext, setContext } = utils.createContextHandlers<{
		form: SuperForm<t.GenericRecord>;
	}>('form');
	export { getContext };
</script>

<script lang="ts" generics="T extends t.GenericRecord">
	import type { Snippet } from 'svelte';
	import type { HTMLFormAttributes } from 'svelte/elements';

	import { form as _form } from '#';

	import Alert from '@/components/ui-custom/alert.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import { m } from '@/i18n';

	//

	type Props = {
		// eslint-disable-next-line no-undef
		form: _form.Instance<T>;
		children?: Snippet<[{ formError: typeof formError; submitButton: typeof submitButton }]>;
		hide?: ('error' | 'submit_button')[];
	};

	let { form, children, hide = [] }: Props = $props();

	const superform = form.attachSuperform();
	const { enhance } = superform;

	type Enctype = HTMLFormAttributes['enctype'];
	const enctype: Enctype =
		superform.options.dataType == 'form' ? 'multipart/form-data' : undefined;

	// This is for backward compatibility
	// @ts-expect-error - Slight type mismatch
	setContext({ form: superform });
</script>

<form method="post" use:enhance {enctype}>
	{@render children?.({ formError, submitButton })}

	{#if !hide.includes('error')}
		{@render formError()}
	{/if}

	{#if !hide.includes('submit_button')}
		{@render submitButton()}
	{/if}
</form>

{#snippet formError()}
	{#if form.submitError}
		<Alert variant="destructive">
			{form.submitError.message}
		</Alert>
	{/if}
{/snippet}

{#snippet submitButton(text = m.Submit())}
	<Button type="submit" disabled={form.delayed || !form.valid}>
		{#if !form.delayed}
			{text}
		{:else}
			<Spinner />
			{m.Please_wait()}
		{/if}
	</Button>
{/snippet}
