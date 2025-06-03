// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { type StandardsWithTestSuites } from '$lib/standards';
import type { CustomChecksResponse } from '@/pocketbase/types';
import { SelectTestsForm, type SelectTestsFormData } from './select-tests-form';
import { getChecksConfigFormProps, type ChecksConfigFormProps } from './checks-configs-form';

//

export type StartChecksFormProps = {
	standardsWithTestSuites: StandardsWithTestSuites;
	customChecks: CustomChecksResponse[];
};

export class StartChecksForm {
	public readonly selectTestsForm: SelectTestsForm;
	public checksConfigsFormProps: ChecksConfigFormProps | undefined;

	state: 'select-tests' | 'fill-values' = $state('select-tests');

	isLoadingData = $state(false);
	loadingError = $state<Error>();

	selectedCustomChecksIds = $state<string[]>([]);
	selectedCustomChecks = $derived.by(() =>
		this.props.customChecks.filter((c) => this.selectedCustomChecksIds.includes(c.id))
	);

	constructor(public readonly props: StartChecksFormProps) {
		this.selectTestsForm = new SelectTestsForm({
			standards: props.standardsWithTestSuites,
			customChecks: props.customChecks,
			onSubmit: (data) => this.handleChecksSelection(data)
		});
	}

	private async handleChecksSelection(data: SelectTestsFormData) {
		this.selectedCustomChecksIds = data.customChecks;
		this.isLoadingData = true;
		try {
			const standardAndVersionPath = data.standardId + '/' + data.versionId;
			const standardChecks = await getChecksConfigFormProps(
				standardAndVersionPath,
				data.tests
			);
			this.checksConfigsFormProps = {
				standardAndVersionPath,
				standardChecks,
				customChecks: this.selectedCustomChecks
			};
			this.state = 'fill-values';
		} catch (error) {
			this.loadingError = error as Error;
		} finally {
			this.isLoadingData = false;
		}
	}

	backToSelectTests() {
		this.state = 'select-tests';
		this.checksConfigsFormProps = undefined;
		this.loadingError = undefined;
		this.selectedCustomChecksIds = [];
	}
}
