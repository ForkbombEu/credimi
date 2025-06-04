// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { type StandardsWithTestSuites } from '$lib/standards';
import type { CustomChecksResponse } from '@/pocketbase/types';
import { SelectChecksForm, type SelectChecksSubmitData } from './select-checks-form';
import { type ConfigureChecksFormProps } from './configure-checks-form';

//

export type StartChecksFormProps = {
	standardsWithTestSuites: StandardsWithTestSuites;
	customChecks: CustomChecksResponse[];
};

export class StartChecksForm {
	public readonly selectChecksForm: SelectChecksForm;
	configureChecksFormProps = $state<ConfigureChecksFormProps>();

	state: 'select-tests' | 'fill-values' = $state('select-tests');

	//

	constructor(public readonly props: StartChecksFormProps) {
		this.selectChecksForm = new SelectChecksForm({
			standards: props.standardsWithTestSuites,
			customChecks: props.customChecks,
			onSubmit: (data) => this.handleChecksSelection(data)
		});
	}

	private async handleChecksSelection(data: SelectChecksSubmitData) {
		this.configureChecksFormProps = data;
		this.state = 'fill-values';
	}

	backToSelectTests() {
		this.state = 'select-tests';
		this.configureChecksFormProps = undefined;
	}
}
