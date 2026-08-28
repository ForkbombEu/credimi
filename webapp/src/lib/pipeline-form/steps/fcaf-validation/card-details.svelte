<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { CardDetailsComponentProps } from '$pipeline-form/steps';

	import { m } from '@/i18n';

	import {
		getFCAFValidationTestIDs,
		type FCAFValidationFormData
	} from './fcaf-validation-step-form.svelte.js';

	//

	let { data }: CardDetailsComponentProps<FCAFValidationFormData> = $props();

	const testIDs = $derived(getFCAFValidationTestIDs(data.yaml));
</script>

{#if testIDs.length > 0}
	<div class="space-y-1.5">
		<p class="text-xs font-medium text-muted-foreground">{m.Tests()}:</p>
		<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
		<div
			aria-label={m.Tests()}
			class="max-h-60 overflow-y-auto rounded-md border bg-muted/30"
			role="region"
			tabindex="0"
		>
			<ul class="divide-y">
				{#each testIDs as testID, index (`${testID}-${index}`)}
					<li class="truncate px-2 py-1 font-mono text-xs" title={testID}>{testID}</li>
				{/each}
			</ul>
		</div>
	</div>
{/if}
