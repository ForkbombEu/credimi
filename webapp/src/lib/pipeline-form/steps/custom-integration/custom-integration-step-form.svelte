<!-- SPDX-FileCopyrightText: 2025 Forkbomb BV -->
<!-- SPDX-License-Identifier: AGPL-3.0-or-later -->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';
	import type { CustomIntegrationStepForm } from './custom-integration-step-form.svelte.js';
	let { self: _form }: SelfProp<CustomIntegrationStepForm> = $props();
</script>
