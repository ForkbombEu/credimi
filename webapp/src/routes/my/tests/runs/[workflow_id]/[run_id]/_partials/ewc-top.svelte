<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { setupEWCConnections } from '$lib/wallet-test-pages/ewc.svelte';

	type Props = {
		workflowId: string;
		namespace: string;
	};

	let { workflowId, namespace }: Props = $props();

	setupEWCConnections(
		() => workflowId,
		() => namespace
	);
</script>
