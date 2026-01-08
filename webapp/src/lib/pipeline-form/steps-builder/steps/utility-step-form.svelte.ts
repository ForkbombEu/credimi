// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { StepFormState, type UtilityStepType, StepType } from '../types.js';

//

type Props<T> = {
	stepType: UtilityStepType;
	onSubmit: (data: T) => void;
};

export class UtilityStepForm<T = any> extends StepFormState {
	constructor(private props: Props<T>) {
		super();
	}

	// Form data for each utility step type
	emailData = $state({
		recipient: '',
		subject: '',
		body: ''
	});

	httpRequestData = $state({
		method: 'GET',
		url: '',
		headers: {},
		body: ''
	});

	get stepType() {
		return this.props.stepType;
	}

	get isEmail() {
		return this.props.stepType === StepType.Email;
	}

	get isDebug() {
		return this.props.stepType === StepType.Debug;
	}

	get isHttpRequest() {
		return this.props.stepType === StepType.HttpRequest;
	}

	canSubmit() {
		if (this.isEmail) {
			return this.emailData.recipient.trim() !== '';
		}
		if (this.isHttpRequest) {
			return this.httpRequestData.url.trim() !== '' && this.httpRequestData.method.trim() !== '';
		}
		// Debug step doesn't need validation
		return true;
	}

	submit() {
		if (!this.canSubmit()) return;

		let data: any = {};
		if (this.isEmail) {
			data = { ...this.emailData };
		} else if (this.isHttpRequest) {
			data = { ...this.httpRequestData };
		} else if (this.isDebug) {
			data = {};
		}

		this.props.onSubmit(data);
	}
}
