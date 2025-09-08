<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Sheet from '@/components/ui-custom/sheet.svelte';

	import { Manager } from './credential-issuer-manager.svelte.js';

	//

	type Props = {
		manager: Manager;
	};

	let { manager }: Props = $props();
</script>

<Sheet bind:open={manager.sheet.isOpen}>ciao</Sheet>
