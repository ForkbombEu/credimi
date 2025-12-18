// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Renderable } from '$lib/renderable';
import { getStandardsWithTestSuites, type StandardsWithTestSuites } from '$lib/standards/index.js';
import { resource } from 'runed';
import { tick } from 'svelte';

//

type Props = {
	onSelect: (checkId: string) => void;
};

export class ConformanceCheckStepForm extends  implements Renderable<ConformanceCheckStepForm> {
	constructor(private props: Props) {
		super();
	}

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
			throw new Error('Invalid state');
		}
	});

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
		this.props.onSelect(test);
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

type FormData = {
	standard: Standard;
	version: Version;
	suite: Suite;
	test: Test;
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
