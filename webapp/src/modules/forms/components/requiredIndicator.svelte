<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { formFieldProxy, type FormPath } from 'sveltekit-superforms';
	import { getFormContext } from '../form.svelte';
	import type { GenericRecord } from '@/utils/types';

	interface Props {
		field: string;
	}

	const { field }: Props = $props();

	const { form, hideRequiredIndicator } = $derived(getFormContext());
	const { constraints } = $derived(formFieldProxy(form, field as FormPath<GenericRecord>));
	const isFieldRequired = $derived($constraints?.required);
</script>

{#if !hideRequiredIndicator && isFieldRequired}
	<span class="text-destructive ml-1 font-bold">*</span>
{/if}
