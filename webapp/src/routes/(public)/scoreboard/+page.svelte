<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Scoreboard } from '$lib';

	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	const scoreboard = new Scoreboard.Instance({
		pageSize: 20
	});
</script>

<div class="grow bg-secondary pt-0 pb-20">
	<div class="mx-auto mb-8 max-w-7xl px-4 pt-10 md:px-8">
		<T tag="h1">{m.Scoreboard()}</T>
	</div>
	<div class="px-4">
		<Scoreboard.Component {scoreboard} />
	</div>
</div>
