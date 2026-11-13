<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { DeviceRecord } from '$lib/pipeline/device/types';

	import { BotIcon } from '@lucide/svelte';
	import { Pipeline } from '$lib';

	import type { OrganizationsResponse, WalletsResponse } from '@/pocketbase/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Dialog from '@/components/ui-custom/dialog.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { goto, m } from '@/i18n';

	import {
		listBuilds,
		listDevices,
		startSession,
		type AuthorDevice,
		type BuildOption
	} from './api';
	import {
		buildLabel,
		buildPlatforms,
		buildsForPlatform,
		preferredDevice,
		toDeviceRecords,
		walletBuilds,
		walletPath
	} from './builds';

	//

	type Props = {
		wallet: WalletsResponse;
		/** The wallet's organization, whose path names the wallet's builds. */
		walletOwner: OrganizationsResponse;
		/** The caller's organization. */
		organization: OrganizationsResponse;
	};

	let { wallet, walletOwner, organization }: Props = $props();

	type Options = { builds: BuildOption[]; devices: AuthorDevice[]; records: DeviceRecord[] };

	let open = $state(false);
	let options = $state<Promise<Options>>();
	let selectedDevice = $state<AuthorDevice>();
	let selectedBuild = $state('');
	let starting = $state(false);
	let startError = $state('');

	// A slower earlier load (after a quick close and reopen) must not overwrite the selection.
	let loadId = 0;

	async function loadOptions(): Promise<Options> {
		const id = ++loadId;
		const [devices, builds] = await Promise.all([listDevices(), listBuilds()]);
		if (devices.isErr) throw new Error(devices.error.message);
		if (builds.isErr) throw new Error(builds.error.message);
		const walletBuildList = walletBuilds(builds.value, walletPath(walletOwner, wallet));
		const platforms = buildPlatforms(walletBuildList);
		if (id === loadId) {
			const preferred = preferredDevice(devices.value, platforms);
			selectedDevice = preferred;
			selectedBuild = preferred
				? (buildsForPlatform(walletBuildList, preferred.platform)[0]?.identifier ?? '')
				: '';
		}
		return {
			builds: walletBuildList,
			devices: devices.value,
			records: toDeviceRecords(devices.value, platforms)
		};
	}

	$effect(() => {
		if (!open) return;
		options = loadOptions();
		startError = '';
	});

	function selectDevice(record: DeviceRecord, opts: Options) {
		selectedDevice = opts.devices.find((d) => d.path === record.path);
		if (!selectedDevice) return;
		selectedBuild =
			buildsForPlatform(opts.builds, selectedDevice.platform)[0]?.identifier ?? '';
	}

	async function start() {
		if (!selectedDevice || !selectedBuild || starting) return;
		starting = true;
		startError = '';
		const res = await startSession(selectedDevice.path, selectedBuild);
		starting = false;
		if (res.isErr) {
			startError = res.error.message;
			return;
		}
		const { session_id, device_id } = res.value;
		open = false;
		await goto(`/my/action-author/${session_id}?device=${encodeURIComponent(device_id)}`);
	}
</script>

<Dialog bind:open title={m.Start_authoring_session()} contentClass="sm:max-w-xl">
	{#snippet trigger({ props })}
		<Button variant="outline" size="sm" {...props}>
			<BotIcon />
			{m.Write_with_the_agent()}
		</Button>
	{/snippet}

	{#snippet content()}
		{#await options}
			<div class="flex items-center justify-center gap-2 py-6">
				<Spinner size={16} />
				<T class="text-muted-foreground">{m.Loading()}</T>
			</div>
		{:then opts}
			{#if opts}
				{#if opts.builds.length === 0}
					<T class="text-sm text-muted-foreground">
						{walletOwner.id === organization.id
							? m.No_authoring_build()
							: m.No_authoring_build_public_wallet()}
					</T>
				{:else if opts.records.length === 0}
					<T class="text-sm text-muted-foreground">{m.No_authoring_device()}</T>
				{:else}
					<div class="space-y-4">
						<div class="max-h-[50dvh] overflow-auto">
							<Pipeline.Device.SelectList
								presentation="run"
								foundDevices={opts.records}
								catalogLoading={false}
								selectedDevice={selectedDevice?.path}
								onSelect={(record) => selectDevice(record, opts)}
							/>
						</div>

						{#if selectedDevice}
							<label class="block space-y-1 text-sm">
								<span class="font-medium">{m.Version()}</span>
								<select
									bind:value={selectedBuild}
									class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
								>
									{#each buildsForPlatform(opts.builds, selectedDevice.platform) as build (build.identifier)}
										<option value={build.identifier}>{buildLabel(build)}</option
										>
									{/each}
								</select>
							</label>
						{/if}

						{#if startError}
							<p class="text-sm text-destructive" role="alert">{startError}</p>
						{/if}

						<div class="flex justify-end">
							<Button
								onclick={start}
								disabled={!selectedDevice || !selectedBuild || starting}
							>
								{#if starting}
									<Spinner size={14} />
								{/if}
								{m.Start_session()}
							</Button>
						</div>
					</div>
				{/if}
			{/if}
		{:catch error}
			<p class="text-sm text-destructive" role="alert">{error.message}</p>
		{/await}
	{/snippet}
</Dialog>
