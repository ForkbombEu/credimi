// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CollectionFormOptions } from './collectionFormTypes';

import CollectionForm from './collectionForm.svelte';
import { removeEmptyValues, setupCollectionForm } from './collectionFormSetup';
import { getCollectionFormContext } from './context';

export {
	CollectionForm,
	getCollectionFormContext,
	removeEmptyValues,
	setupCollectionForm,
	type CollectionFormOptions
};
