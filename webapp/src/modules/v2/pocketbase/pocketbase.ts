// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type * as pb from 'pocketbase';
import type { Simplify } from 'type-fest';

import GenericPocketBase from 'pocketbase';

import { pb as defaultPocketbaseClient } from '@/pocketbase';

//

export type Pocketbase = GenericPocketBase;

export type QueryOptions = Simplify<
	pb.FullListOptions | pb.RecordListOptions | pb.RecordOptions | pb.ListOptions
>;

export const defaultClient = defaultPocketbaseClient;
