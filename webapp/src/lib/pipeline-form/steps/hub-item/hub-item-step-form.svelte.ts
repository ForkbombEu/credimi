// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';
import type { HubItem, HubItemType } from '$lib/hub';

import { BaseForm, type InitFormOptions } from '../types';
import Component from './hub-item-step-form.svelte';

const collections = [
	'credentials',
	'use_cases_verifications',
	'custom_checks'
] as const satisfies HubItemType[];

type HubStepCollection = (typeof collections)[number];

type Props = {
	collection: HubStepCollection;
	entityData: EntityData;
};

export class HubItemStepForm extends BaseForm<HubItem, HubItemStepForm> {
	readonly Component = Component;

	selectedItem = $state<HubItem | undefined>(undefined);

	constructor(
		private props: Props,
		opts?: InitFormOptions<HubItem>
	) {
		super(opts);
		if (opts?.initial) {
			this.selectedItem = opts.initial;
		}
	}

	canSave() {
		return this.selectedItem !== undefined;
	}

	getSubmitData() {
		return this.selectedItem;
	}

	selectItem(item: HubItem) {
		this.selectedItem = item;
		if (this.intent === 'add') {
			this.commit(item);
		}
	}

	discardSelection() {
		this.selectedItem = undefined;
	}

	get collection() {
		return this.props.collection;
	}

	get entityData() {
		return this.props.entityData;
	}
}
