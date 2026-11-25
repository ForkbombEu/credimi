<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { DeviceRecord } from '$lib/pipeline/device/types';

	import { PlayIcon } from '@lucide/svelte';
	import { Pipeline } from '$lib';

	import type { OrganizationsResponse, WalletsResponse } from '@/pocketbase/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Spinner from '@/components/ui-custom/spinner.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import {
		listBuilds,
		listDevices,
		runFcafPipeline,
		type AuthorDevice,
		type BuildOption,
		type FcafPipelineRun
	} from './api';
	import { buildLabel, buildsForPlatform, walletBuilds, walletPath } from './builds';
	import { fcafDeviceRecords, fcafPreferredDevice, pipelineRunLink } from './fcaf';

	//

	type Props = {
		wallet: Pick<WalletsResponse, 'canonified_name'>;
		/** The wallet's organization, whose path names the wallet's builds. */
		walletOwner: Pick<OrganizationsResponse, 'canonified_name'>;
	};

	let { wallet, walletOwner }: Props = $props();

	type Options = { builds: BuildOption[]; devices: AuthorDevice[] };

	let expanded = $state(false);
	let options = $state<Promise<Options>>();
	let selectedDevice = $state<AuthorDevice>();
	let selectedBuild = $state('');
	let running = $state(false);
	let runError = $state('');
	let started = $state<FcafPipelineRun>();

	// A slower earlier load (after a quick close and reopen) must not overwrite the selection.
	let loadId = 0;

	async function loadOptions(): Promise<Options> {
		const id = ++loadId;
		const [devices, builds] = await Promise.all([listDevices(), listBuilds()]);
		if (devices.isErr) throw new Error(devices.error.message);
		if (builds.isErr) throw new Error(builds.error.message);
		const androidBuilds = buildsForPlatform(
			walletBuilds(builds.value, walletPath(walletOwner, wallet)),
			'android'
		);
		if (id === loadId) {
			selectedDevice = fcafPreferredDevice(devices.value);
			selectedBuild = androidBuilds[0]?.identifier ?? '';
		}
		return { builds: androidBuilds, devices: devices.value };
	}

	function toggle() {
		expanded = !expanded;
		if (!expanded) return;
		runError = '';
		started = undefined;
		options = loadOptions();
	}

	function selectDevice(record: DeviceRecord, opts: Options) {
		selectedDevice = opts.devices.find((d) => d.path === record.path);
	}

	async function run() {
		if (!selectedDevice || !selectedBuild || running) return;
		running = true;
		runError = '';
		const res = await runFcafPipeline(selectedBuild, selectedDevice.path);
		running = false;
		if (res.isErr) {
			runError = res.error.message;
			return;
		}
		started = res.value;
	}
</script>

<div class="space-y-3">
	<Button variant="outline" size="sm" onclick={toggle} aria-expanded={expanded}>
		<PlayIcon />
		{m.Fcaf_run_validation()}
	</Button>

	{#if expanded}
		<div class="space-y-3 rounded-md border p-3 text-sm">
			<p class="text-xs text-muted-foreground">{m.Fcaf_run_validation_description()}</p>

			{#await options}
				<div class="flex items-center justify-center gap-2 py-4">
					<Spinner size={16} />
					<T class="text-muted-foreground">{m.Loading()}</T>
				</div>
			{:then opts}
				{#if opts}
					{@const records = fcafDeviceRecords(opts.devices)}
					{#if opts.builds.length === 0}
						<T class="text-sm text-muted-foreground">{m.Fcaf_run_no_build()}</T>
					{:else if records.length === 0}
						<T class="text-sm text-muted-foreground">{m.Fcaf_run_no_device()}</T>
					{:else}
						<div class="max-h-[40dvh] overflow-auto">
							<Pipeline.Device.SelectList
								presentation="run"
								foundDevices={records}
								catalogLoading={false}
								selectedDevice={selectedDevice?.path}
								onSelect={(record) => selectDevice(record, opts)}
							/>
						</div>

						<label class="block space-y-1">
							<span class="font-medium">{m.Version()}</span>
							<select
								bind:value={selectedBuild}
								class="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
							>
								{#each opts.builds as build (build.identifier)}
									<option value={build.identifier}>{buildLabel(build)}</option>
								{/each}
							</select>
						</label>

						{#if runError}
							<p class="text-destructive" role="alert">{runError}</p>
						{/if}

						<div class="flex justify-end">
							<Button
								size="sm"
								onclick={run}
								disabled={!selectedDevice || !selectedBuild || running}
							>
								{#if running}
									<Spinner size={14} />
								{/if}
								{m.Fcaf_run_start()}
							</Button>
						</div>

						{#if started}
							<div class="space-y-2 border-t pt-3" role="status">
								<p>
									{m.Fcaf_run_queued({
										count: started.test_count,
										status: started.status
									})}
								</p>
								<a
									href={pipelineRunLink(started)}
									target="_blank"
									rel="noreferrer"
									class="text-primary hover:underline"
								>
									{m.Fcaf_run_see()}
								</a>
								{#if started.skipped.length}
									<div class="space-y-1">
										<p class="font-medium">{m.Fcaf_run_skipped()}</p>
										<ul class="space-y-1">
											{#each started.skipped as skipped (skipped.scenario)}
												<li class="text-xs">
													<span class="font-mono break-all"
														>{skipped.scenario}</span
													>
													<span
														class="block font-mono break-all text-muted-foreground"
													>
														{m.Fcaf_run_missing_roles({
															roles: skipped.missing_roles.join(', ')
														})}
													</span>
												</li>
											{/each}
										</ul>
									</div>
								{/if}
							</div>
						{/if}
					{/if}
				{/if}
			{:catch error}
				<p class="text-destructive" role="alert">{error.message}</p>
			{/await}
		</div>
	{/if}
</div>
