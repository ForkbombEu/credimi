// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getStandardsWithTestSuites, type StandardsWithTestSuites } from '$lib/standards/index.js';
import { getPath } from '$lib/utils';
import { resource } from 'runed';
import { tick } from 'svelte';

import type { WalletActionsResponse } from '@/pocketbase/types';

import { m } from '@/i18n';
import { pb } from '@/pocketbase';

import { ExecutionTarget } from '../../execution-target';
import { BaseForm, type InitFormOptions } from '../types';
import Component from './conformance-check-step-form.svelte';

//

const OPENID4VCI_WALLET_ACTION_CATEGORY = 'get-credential-generic';

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
			return 'ready';
		} else {
			throw new Error(m.Pipeline_form_invalid_state());
		}
	});

	//

	availableVersions = $derived(this.data.standard?.versions ?? []);
	availableSuites = $derived(this.data.version?.suites ?? []);
	availableTests = $derived(this.data.suite?.paths ?? []);

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
			await this.selectTest(this.availableTests[0]);
		}
	}

	async selectTest(test: Test) {
		const action_id = test.startsWith('openid4vci_wallet')
			? await getOpenID4VCIWalletActionId()
			: undefined;

		this.data.test = test;
		this.data.action_id = action_id;
		if (this.intent === 'add') {
			this.commit({ ...this.data, test, action_id } as FormData);
		}
	}

	discardTest() {
		this.data.test = undefined;
	}

	//

	discardSuite() {
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

export type FormState =
	| 'select-standard'
	| 'select-version'
	| 'select-suite'
	| 'select-test'
	| 'ready'
	| 'loading'
	| 'error';

//

type Standard = StandardsWithTestSuites[number];
type Version = Standard['versions'][number];
type Suite = Version['suites'][number];
type Test = Suite['paths'][number];

async function getOpenID4VCIWalletActionId() {
	const wallet = ExecutionTarget.state.current?.wallet;
	if (!wallet) {
		throw new Error(m.Pipeline_form_choose_wallet_before_openid4vci_wallet_check());
	}
	const actions = await pb.collection('wallet_actions').getFullList<WalletActionsResponse>({
		filter: pb.filter('wallet = {:wallet} && category ~ {:category}', {
			wallet: wallet.id,
			category: OPENID4VCI_WALLET_ACTION_CATEGORY
		}),
		sort: 'created'
	});

	const action = actions.find((action) => action.category === OPENID4VCI_WALLET_ACTION_CATEGORY);

	if (!action) {
		throw new Error(
			m.Pipeline_form_wallet_missing_action_category({
				wallet: wallet.name,
				category: OPENID4VCI_WALLET_ACTION_CATEGORY
			})
		);
	}

	return getPath(action);
}
