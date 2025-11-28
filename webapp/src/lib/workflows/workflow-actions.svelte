<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { WorkflowStatus } from '@forkbombeu/temporal-ui/dist/types/workflows';
	import type { ClassValue } from 'svelte/elements';

	import { Code, Hourglass, XIcon } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	import { cancelWorkflow } from './utils';

	//

	type Props = {
		execution: { workflowId: string; runId: string };
		status: WorkflowStatus;
		containerClass?: ClassValue;
	};

	let { execution, status, containerClass }: Props = $props();
</script>

<div class={['flex gap-2', containerClass]}>
	<Button
		variant="outline"
		onclick={() => cancelWorkflow(execution)}
		disabled={status !== 'Running'}
		size="sm"
	>
		<XIcon />
		{m.Terminate()}
	</Button>
	<Button disabled variant="outline" size="sm">
		<Hourglass />
		{m.Schedule()}
	</Button>
	<Button disabled variant="outline" size="sm">
		<Code />
		{m.Swagger()}
	</Button>
</div>
