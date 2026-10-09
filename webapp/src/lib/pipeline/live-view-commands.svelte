<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Component } from 'svelte';

	import {
		CircleCheckIcon,
		CircleDashedIcon,
		CircleSlashIcon,
		CircleXIcon,
		LoaderCircleIcon,
		TriangleAlertIcon
	} from '@lucide/svelte';
	import { tick } from 'svelte';

	import { m } from '@/i18n';

	import {
		fetchLiveViewCommands,
		groupMaestroCommands,
		mergeMaestroCommands,
		type MaestroCommand,
		type MaestroCommandStatus
	} from './live-view';

	type Props = { streamUrl: string };

	let { streamUrl }: Props = $props();

	const RETRY_DELAY_MS = 2000;
	const STICK_TO_BOTTOM_PX = 16;

	const STATUS_ICONS: Record<MaestroCommandStatus, { icon: Component; class: string }> = {
		RUNNING: { icon: LoaderCircleIcon, class: 'animate-spin' },
		COMPLETED: { icon: CircleCheckIcon, class: 'text-green-600' },
		FAILED: { icon: CircleXIcon, class: 'text-red-600' },
		WARNED: { icon: TriangleAlertIcon, class: 'text-amber-500' },
		SKIPPED: { icon: CircleSlashIcon, class: 'text-muted-foreground' },
		PENDING: { icon: CircleDashedIcon, class: 'text-muted-foreground' }
	};

	let commands = $state<MaestroCommand[]>([]);
	let ended = $state(false);
	let container = $state<HTMLElement>();

	const groups = $derived(groupMaestroCommands(commands));

	$effect(() => {
		const controller = new AbortController();
		poll(streamUrl, controller.signal).catch(() => {
			// Aborted on unmount or stream change.
		});
		return () => controller.abort();
	});

	async function poll(url: string, signal: AbortSignal) {
		let after = 0;
		while (!signal.aborted) {
			const page = await fetchLiveViewCommands(url, after, signal);
			if (page.isOk) {
				await apply(page.value.commands);
				after = page.value.next;
			} else if (page.error === 404) {
				ended = true;
				return;
			} else {
				await sleep(RETRY_DELAY_MS, signal);
			}
		}
	}

	async function apply(incoming: MaestroCommand[]) {
		if (incoming.length === 0) return;
		const stick =
			container !== undefined &&
			container.scrollHeight - container.scrollTop - container.clientHeight <=
				STICK_TO_BOTTOM_PX;
		commands = mergeMaestroCommands(commands, incoming);
		if (!stick) return;
		await tick();
		if (container) container.scrollTop = container.scrollHeight;
	}

	function sleep(ms: number, signal: AbortSignal): Promise<void> {
		return new Promise((resolve) => {
			const timer = setTimeout(resolve, ms);
			signal.addEventListener(
				'abort',
				() => {
					clearTimeout(timer);
					resolve();
				},
				{ once: true }
			);
		});
	}
</script>

<section class="flex min-w-0 flex-col gap-2">
	<p class="text-sm font-semibold">{m.Maestro_commands()}</p>
	<div bind:this={container} class="max-h-[70dvh] space-y-3 overflow-y-auto">
		{#if commands.length === 0 && !ended}
			<p class="text-sm text-muted-foreground">{m.Maestro_commands_waiting()}</p>
		{:else}
			{#each groups as group, i (i)}
				<div class="space-y-1">
					<p class="text-xs font-semibold">{m.Maestro_flow_number({ number: i + 1 })}</p>
					<ol class="space-y-1">
						{#each group.commands as c (c.index)}
							{@const status = STATUS_ICONS[c.status]}
							<li
								class="flex items-start gap-2 font-mono text-xs"
								style:padding-left="{c.depth}rem"
							>
								<status.icon class="size-3.5 shrink-0 {status.class}" />
								<span class="break-words">{c.description}</span>
							</li>
						{/each}
					</ol>
				</div>
			{/each}
		{/if}
	</div>
</section>
