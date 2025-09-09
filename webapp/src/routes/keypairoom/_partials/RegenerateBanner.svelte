<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TriangleAlert } from 'lucide-svelte';

	import Alert from '@/components/ui-custom/alert.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
</script>

<Alert variant="warning" icon={TriangleAlert}>
	<div class="space-y-1">
		<T>{m.You_have_been_redirected_here_because_your_private_keys_are_missing_()}</T>
		<T>{m.Before_using_the_app_again_you_need_to_restore_them_()}</T>
	</div>
</Alert>
