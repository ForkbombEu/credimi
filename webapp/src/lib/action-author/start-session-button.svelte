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
	import { Checkbox } from '@/components/ui/checkbox';
	import { Label } from '@/components/ui/label';
	import * as RadioGroup from '@/components/ui/radio-group/index.js';
	import { Textarea } from '@/components/ui/textarea';
	import { goto, m } from '@/i18n';

	import {
		getFcafCoverage,
		listBuilds,
		listDevices,
		startSession,
		type AuthorDevice,
		type AuthoringMode,
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
	import { fcafGuidance, fixableRoles, modePlatforms } from './fcaf';

	//

	type Props = {
		wallet: WalletsResponse;
		/** The wallet's organization, whose path names the wallet's builds. */
		walletOwner: OrganizationsResponse;
		/** The caller's organization. */
		organization: OrganizationsResponse;
	};

	let { wallet, walletOwner, organization }: Props = $props();

	type Options = { builds: BuildOption[]; devices: AuthorDevice[] };

	let open = $state(false);
	let options = $state<Promise<Options>>();
	let mode = $state<AuthoringMode>('chat');
	let selectedDevice = $state<AuthorDevice>();
	let selectedBuild = $state('');
	let starting = $state(false);
	let startError = $state('');
	let guidance = $state('');
	let fcafFix = $state(false);
	/** The wallet's roles that failed in its last FCAF run; 0 until its coverage loads, or when that fails. */
	let fixableCount = $state(0);

	// A slower earlier coverage load must not show the option in another mode or dialog.
	let coverageId = 0;

	async function loadFixable() {
		const id = ++coverageId;
		const res = await getFcafCoverage(walletPath(walletOwner, wallet));
		if (id === coverageId && res.isOk) fixableCount = fixableRoles(res.value.roles).length;
	}

	// A slower earlier load (after a quick close and reopen) must not overwrite the selection.
	let loadId = 0;

	async function loadOptions(): Promise<Options> {
		const id = ++loadId;
		const [devices, builds] = await Promise.all([listDevices(), listBuilds()]);
		if (devices.isErr) throw new Error(devices.error.message);
		if (builds.isErr) throw new Error(builds.error.message);
		const walletBuildList = walletBuilds(builds.value, walletPath(walletOwner, wallet));
		if (id === loadId) {
			// The mode is always chat when the options load.
			const preferred = preferredDevice(devices.value, buildPlatforms(walletBuildList));
			selectedDevice = preferred;
			selectedBuild = preferred
				? (buildsForPlatform(walletBuildList, preferred.platform)[0]?.identifier ?? '')
				: '';
		}
		return { builds: walletBuildList, devices: devices.value };
	}

	$effect(() => {
		if (!open) return;
		mode = 'chat';
		guidance = '';
		fcafFix = false;
		fixableCount = 0;
		coverageId++;
		options = loadOptions();
		startError = '';
	});

	function setMode(next: AuthoringMode, opts: Options) {
		mode = next;
		fcafFix = false;
		fixableCount = 0;
		coverageId++;
		if (next === 'fcaf') void loadFixable();
		selectedDevice = preferredDevice(
			opts.devices,
			modePlatforms(buildPlatforms(opts.builds), next)
		);
		selectedBuild = selectedDevice
			? (buildsForPlatform(opts.builds, selectedDevice.platform)[0]?.identifier ?? '')
			: '';
	}

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
		const res = await startSession(
			selectedDevice.path,
			selectedBuild,
			mode,
			fcafFix && fixableCount > 0
		);
		starting = false;
		if (res.isErr) {
			startError = res.error.message;
			return;
		}
		const { session_id, device_id } = res.value;
		if (mode === 'fcaf') {
			try {
				fcafGuidance(session_id).start({ text: guidance.trim() });
			} catch {
				/* storage unavailable: the page sends the default first message */
			}
		}
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
				{:else}
					{@const platforms = modePlatforms(buildPlatforms(opts.builds), mode)}
					{@const records = toDeviceRecords(opts.devices, platforms)}
					<div class="space-y-4">
						<div class="space-y-2">
							<Label>{m.Authoring_mode()}</Label>
							<RadioGroup.Root
								bind:value={() => mode, (v) => setMode(v as AuthoringMode, opts)}
								aria-label={m.Authoring_mode()}
							>
								<div class="flex items-start gap-2">
									<RadioGroup.Item
										value="chat"
										id="{wallet.id}-mode-chat"
										class="mt-0.5"
									/>
									<div class="space-y-0.5">
										<Label for="{wallet.id}-mode-chat"
											>{m.Authoring_mode_chat()}</Label
										>
										<p class="text-xs text-muted-foreground">
											{m.Authoring_mode_chat_description()}
										</p>
									</div>
								</div>
								<div class="flex items-start gap-2">
									<RadioGroup.Item
										value="fcaf"
										id="{wallet.id}-mode-fcaf"
										class="mt-0.5"
									/>
									<div class="space-y-0.5">
										<Label for="{wallet.id}-mode-fcaf"
											>{m.Authoring_mode_fcaf()}</Label
										>
										<p class="text-xs text-muted-foreground">
											{m.Authoring_mode_fcaf_description()}
										</p>
									</div>
								</div>
							</RadioGroup.Root>
						</div>

						{#if mode === 'fcaf' && fixableCount > 0}
							<div class="flex items-start gap-2">
								<Checkbox
									id="{wallet.id}-fcaf-fix"
									bind:checked={fcafFix}
									class="mt-0.5"
								/>
								<div class="space-y-0.5">
									<Label for="{wallet.id}-fcaf-fix">{m.Fcaf_fix_failing()}</Label>
									<p class="text-xs text-muted-foreground">
										{m.Fcaf_fix_failing_description({ count: fixableCount })}
									</p>
								</div>
							</div>
						{/if}

						{#if mode === 'fcaf'}
							<label class="block space-y-1 text-sm">
								<span class="font-medium">{m.Fcaf_guidance()}</span>
								<Textarea
									bind:value={guidance}
									maxlength={2000}
									rows={3}
									placeholder={m.Fcaf_guidance_placeholder()}
								/>
							</label>
						{/if}

						{#if mode === 'fcaf' && platforms.size === 0}
							<T class="text-sm text-muted-foreground"
								>{m.No_fcaf_authoring_build()}</T
							>
						{:else if records.length === 0}
							<T class="text-sm text-muted-foreground">{m.No_authoring_device()}</T>
						{:else}
							<div class="max-h-[50dvh] overflow-auto">
								<Pipeline.Device.SelectList
									presentation="run"
									foundDevices={records}
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
											<option value={build.identifier}
												>{buildLabel(build)}</option
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
						{/if}
					</div>
				{/if}
			{/if}
		{:catch error}
			<p class="text-sm text-destructive" role="alert">{error.message}</p>
		{/await}
	{/snippet}
</Dialog>
