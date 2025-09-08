<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { form as f, pocketbase as pb } from '#';

	type Props = {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		form: pb.recordform.Instance<any>;
	};

	let { form }: Props = $props();
</script>

{#key form.currentMode}
	{#if form.form}
		<f.Component form={form.form} />
	{/if}
{/key}
