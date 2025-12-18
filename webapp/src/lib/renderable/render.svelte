<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Renderable, SelfProps } from './types';

	let { self }: SelfProps<Renderable> = $props();
</script>

<self.Component {self} />
