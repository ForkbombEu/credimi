<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TriangleAlert } from 'lucide-svelte';

	import Alert from '@/components/ui-custom/alert.svelte';
	import { getFormContext } from '@/forms';
	import { m } from '@/i18n';

	import { FORM_ERROR_PATH } from '../form';

	//

	const { form } = $derived(getFormContext());
	const { allErrors } = $derived(form);
	const error = $derived($allErrors.find((e) => e.path == FORM_ERROR_PATH));
</script>

{#if error}
	<Alert variant="destructive" icon={TriangleAlert}>
		{#snippet content({ Title, Description })}
			<Title>{m.Error()}</Title>
			<Description>
				{#if error.messages.length > 0}
					{#each error.messages as message}
						{message}
					{/each}
				{/if}
			</Description>
		{/snippet}
	</Alert>
{/if}
