// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import { getStandardsWithTestSuites, type StandardsWithTestSuites } from '$lib/standards/index.js';
import { getPath } from '$lib/utils';
import { resource } from 'runed';
import { tick } from 'svelte';

import { m } from '@/i18n';
import { pb } from '@/pocketbase';
import { WalletActionsCategoryOptions, type WalletActionsResponse } from '@/pocketbase/types';

import { ExecutionTarget } from '../../execution-target';
import { BaseForm, type InitFormOptions } from '../types';
import Component from './conformance-check-step-form.svelte';

//

const OPENID4VCI_WALLET_ACTION_CATEGORY = WalletActionsCategoryOptions['get-credential-generic'];

export class ConformanceCheckStepForm extends BaseForm<FormData, ConformanceCheckStepForm> {
	readonly Component = Component;

	standardsWithTestSuites = resource(
		() => {},
		async () => {
			const result = await getStandardsWithTestSuites({ forPipeline: true });
			if (result instanceof Error) throw result;
			return result;
		},
		{}
	);

	walletActions = resource(
		() => ExecutionTarget.state.current?.wallet?.id,
		async (walletId) => {
			if (!walletId) return null;

			return pb.collection('wallet_actions').getFullList<WalletActionsResponse>({
				filter: pb.filter('wallet = {:wallet} && category ~ {:category}', {
					wallet: walletId,
					category: OPENID4VCI_WALLET_ACTION_CATEGORY
				}),
				sort: 'created'
			});
		},
		{}
	);

	data = $state<Partial<FormData>>({});

	constructor(opts?: InitFormOptions<FormData>) {
		super(opts);
		if (opts?.initial) {
			this.data = { ...opts.initial };
		}
	}

	canSave() {
		return this.state === 'ready';
	}

	getSubmitData() {
		return this.state === 'ready' ? (this.data as FormData) : undefined;
	}

	state: FormState = $derived.by(() => {
		const { standard, version, suite, test } = this.data;
		if (this.standardsWithTestSuites.loading) {
			return 'loading';
		} else if (this.standardsWithTestSuites.error) {
			return 'error';
		} else if (!standard) {
			return 'select-standard';
		} else if (standard && !version) {
			return 'select-version';
		} else if (standard && version && !suite) {
			return 'select-suite';
		} else if (standard && version && suite && !test) {
			return 'select-test';
		} else if (standard && version && suite && test) {
			if (
				isOpenId4VciWalletTest(test) &&
				resolveWalletActionSelection(this.genericCredentialActions).kind === 'picker' &&
				!this.data.action_id
			) {
				return 'select-wallet-action';
			}
			return 'ready';
		} else {
			throw new Error(m.Pipeline_form_invalid_state());
		}
	});

	//

	availableVersions = $derived(this.data.standard?.versions ?? []);
	availableSuites = $derived(this.data.version?.suites ?? []);
	availableTests = $derived(this.data.suite?.paths ?? []);

	hasWalletTests = $derived(this.availableTests.some((test) => isOpenId4VciWalletTest(test)));

	genericCredentialActions = $derived(this.walletActions.current ?? []);

	selectedWalletAction = $derived.by(() => {
		if (!this.data.action_id) return undefined;
		return this.genericCredentialActions.find(
			(action) => getPath(action) === this.data.action_id
		);
	});

	testPickerNotice: TestPickerNotice = $derived.by(() => {
		if (!this.hasWalletTests) {
			return { kind: 'none' };
		}

		const wallet = ExecutionTarget.state.current?.wallet;

		if (wallet && this.walletActions.loading) {
			return { kind: 'loading' };
		}

		const message = getWalletTestBlockReason(wallet, this.walletActions);
		if (message) {
			return { kind: 'alert', message };
		}

		return { kind: 'none' };
	});

	testOptions: TestOption[] = $derived.by(() => {
		const wallet = ExecutionTarget.state.current?.wallet;
		const walletTestsBlocked =
			this.hasWalletTests &&
			(!wallet ||
				this.walletActions.loading ||
				getWalletTestBlockReason(wallet, this.walletActions));

		return this.availableTests.map((test) => {
			const testName = test.split('/').at(-1) ?? test;

			if (!isOpenId4VciWalletTest(test)) {
				return { test, testName, enabled: true };
			}

			if (walletTestsBlocked) {
				return { test, testName, enabled: false };
			}

			return { test, testName, enabled: true };
		});
	});

	async selectStandard(standard: Standard) {
		this.data.standard = standard;
		await tick();
		if (this.availableVersions?.length === 1) {
			await this.selectVersion(this.availableVersions[0]);
		}
	}

	async selectVersion(version: Version) {
		this.data.version = version;
		await tick();
		if (this.availableSuites?.length === 1) {
			await this.selectSuite(this.availableSuites[0]);
		}
	}

	async selectSuite(suite: Suite) {
		this.data.suite = suite;
		await tick();
		if (this.availableTests?.length === 1) {
			const option = this.testOptions[0];
			if (option?.enabled) {
				this.selectTest(option);
			}
		}
	}

	selectTest(option: TestOption) {
		if (!option.enabled) return;

		this.data.test = option.test;

		if (!isOpenId4VciWalletTest(option.test)) {
			this.data.action_id = undefined;
			if (this.intent === 'add') {
				this.commit({ ...this.data, test: option.test } as FormData);
			}
			return;
		}

		const selection = resolveWalletActionSelection(this.genericCredentialActions);
		if (selection.kind === 'auto') {
			this.data.action_id = getPath(selection.action);
			if (this.intent === 'add') {
				this.commit({
					...this.data,
					test: option.test,
					action_id: this.data.action_id
				} as FormData);
			}
		} else {
			this.data.action_id = undefined;
		}
	}

	selectWalletAction(action: WalletActionsResponse) {
		this.data.action_id = getPath(action);
		if (this.intent === 'add') {
			this.commit({ ...this.data, action_id: this.data.action_id } as FormData);
		}
	}

	discardTest() {
		this.data.test = undefined;
		this.data.action_id = undefined;
	}

	discardWalletAction() {
		this.data.action_id = undefined;
		const selection = resolveWalletActionSelection(this.genericCredentialActions);
		if (selection.kind === 'auto') {
			this.data.action_id = getPath(selection.action);
		}
	}

	//

	discardSuite() {
		this.discardTest();
		this.data.suite = undefined;
	}

	discardVersion() {
		this.discardSuite();
		this.data.version = undefined;
	}

	discardStandard() {
		this.discardVersion();
		this.data.standard = undefined;
	}
}

//

export type FormData = {
	standard: Standard;
	version: Version;
	suite: Suite;
	test: Test;
	action_id?: string;
};

export type TestOption = {
	test: Test;
	testName: string;
	enabled: boolean;
};

export type TestPickerNotice =
	| { kind: 'none' }
	| { kind: 'loading' }
	| { kind: 'alert'; message: string };

export type FormState =
	| 'select-standard'
	| 'select-version'
	| 'select-suite'
	| 'select-test'
	| 'select-wallet-action'
	| 'ready'
	| 'loading'
	| 'error';

//

type Standard = StandardsWithTestSuites[number];
type Version = Standard['versions'][number];
type Suite = Version['suites'][number];
type Test = Suite['paths'][number];

export function getWalletTestBlockReason(
	wallet: HubItem | undefined,
	walletActions: {
		loading: boolean;
		error: Error | undefined;
		current: WalletActionsResponse[] | null | undefined;
	}
): string | null {
	if (!wallet) {
		return m.Pipeline_form_choose_wallet_before_openid4vci_wallet_check({
			category: OPENID4VCI_WALLET_ACTION_CATEGORY
		});
	}

	if (walletActions.loading) {
		return null;
	}

	if (walletActions.error) {
		return walletActions.error.message;
	}

	const actions = walletActions.current ?? [];

	if (actions.length === 0) {
		return m.Pipeline_form_wallet_missing_action_category({
			wallet: wallet.name,
			category: OPENID4VCI_WALLET_ACTION_CATEGORY
		});
	}

	return null;
}

export type WalletActionSelection =
	| { kind: 'blocked' }
	| { kind: 'auto'; action: WalletActionsResponse }
	| { kind: 'picker' };

export function isOpenId4VciWalletTest(test: string) {
	return test.startsWith('openid4vci_wallet');
}

export function resolveWalletActionSelection(
	actions: WalletActionsResponse[]
): WalletActionSelection {
	if (actions.length === 0) return { kind: 'blocked' };
	if (actions.length === 1) return { kind: 'auto', action: actions[0]! };
	return { kind: 'picker' };
}
