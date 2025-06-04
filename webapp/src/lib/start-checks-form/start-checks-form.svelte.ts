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
	customCheckId?: string;
};

export class StartChecksForm {
	public readonly selectChecksForm: SelectChecksForm;
	configureChecksFormProps = $state<ConfigureChecksFormProps>();

	state: 'select-tests' | 'fill-values' = $state('select-tests');

	//

	constructor(public readonly props: StartChecksFormProps) {
		const { standardsWithTestSuites, customChecks, customCheckId } = props;
		this.selectChecksForm = new SelectChecksForm({
			standards: standardsWithTestSuites,
			customChecks,
			onSubmit: (data) => this.handleChecksSelection(data)
		});

		const customCheck = customChecks.find((check) => check.id === customCheckId);
		if (customCheck) {
			this.configureChecksFormProps = {
				standardAndVersionPath: customCheck.standard_and_version,
				configsFields: {
					normalized_fields: [],
					specific_fields: {}
				},
				customChecks: [customCheck]
			};
			this.state = 'fill-values';
		}
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
