// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { ZodError } from 'zod';

import { pb } from '@/pocketbase';

import { nestSuites } from './nest';
import {
	compileCheckListQuery,
	compileSuiteListQuery,
	type CheckListIntent,
	type SuiteFacets,
	type SuiteListIntent
} from './query';
import {
	CONFORMANCE_CHECKS_COLLECTION,
	CONFORMANCE_SUITES_COLLECTION,
	conformanceCheckRecordSchema,
	conformanceSuiteRecordSchema,
	type ConformanceCheckRecord,
	type ConformanceSuiteRecord,
	type TemplateSurface
} from './record';
import { standardSchema, type Standard } from './types';

export type { SuiteFacets } from './query';

export type ListChecksError = ClientResponseError | ZodError;
export type ListSuitesError = ClientResponseError | ZodError;

export type ListChecksOptions = CheckListIntent & {
	fetch?: typeof fetch;
};

export type ListSuitesOptions = SuiteListIntent & {
	fetch?: typeof fetch;
};

/**
 * Shared conformance catalog client: flat checks, suite rows, and nested browse
 * tree — PocketBase adapter over compiled domain intents.
 */
export function listChecks(
	options: ListChecksOptions = {}
): Task.Task<ConformanceCheckRecord[], ListChecksError> {
	const { fetch: fetchFn = fetch, ...intent } = options;
	const compiled = compileCheckListQuery(intent, (raw, params) => pb.filter(raw, params));

	const listOptions: {
		fetch: typeof fetch;
		sort: string;
		filter?: string;
	} = {
		fetch: fetchFn,
		sort: compiled.sort,
		...(compiled.filter ? { filter: compiled.filter } : {})
	};

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => pb.collection(CONFORMANCE_CHECKS_COLLECTION).getFullList(listOptions)
	).andThen((rows) => {
		const parsed = conformanceCheckRecordSchema.array().safeParse(rows);
		if (parsed.success) return Task.resolve(parsed.data);
		return Task.reject(parsed.error);
	});
}

/**
 * Suite-grain catalog for the hub table.
 * Default sort intent: wallet→issuer→verifier, then standard, then suite uid.
 */
export function listSuites(
	options: ListSuitesOptions = {}
): Task.Task<ConformanceSuiteRecord[], ListSuitesError> {
	const { fetch: fetchFn = fetch, ...intent } = options;
	const compiled = compileSuiteListQuery(intent, (raw, params) => pb.filter(raw, params));

	const listOptions: {
		fetch: typeof fetch;
		sort: string;
		filter?: string;
	} = {
		fetch: fetchFn,
		sort: compiled.sort,
		...(compiled.filter ? { filter: compiled.filter } : {})
	};

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() => pb.collection(CONFORMANCE_SUITES_COLLECTION).getFullList(listOptions)
	).andThen((rows) => {
		const parsed = conformanceSuiteRecordSchema.array().safeParse(rows);
		if (parsed.success) return Task.resolve(parsed.data);
		return Task.reject(parsed.error);
	});
}

// --- Browse tree (nested standards → versions → suites) ---

export type ListAllResponse = Standard[];
export type ListAllError = ClientResponseError | ZodError;
export type StandardsWithTestSuites = ListAllResponse;

export type ListAllOptions = {
	fetch?: typeof fetch;
	/** Catalog surface — required; no silent default (ADR-0004). */
	surface: TemplateSurface;
	facets?: SuiteFacets;
};

/**
 * Package-internal nest compose (listSuites → nestSuites). Not barrel-exported
 * (ADR-0004): client callers use Store.load; SSR / non-hydrating reads use
 * {@link getStandardsWithTestSuites}.
 */
export function listAll(options: ListAllOptions): Task.Task<ListAllResponse, ListAllError> {
	const { fetch: fetchFn = fetch, surface, facets } = options;

	return listSuites({ fetch: fetchFn, surface, facets }).andThen((records) => {
		const nested = nestSuites(records);
		const res = standardSchema.array().safeParse(nested);
		if (res.success) return Task.resolve(res.data);
		return Task.reject(res.error);
	});
}

/**
 * Sole SSR / non-hydrating nest one-shot (ADR-0004). Catalog surface is
 * required at the call site. Does not hydrate Store.
 */
export async function getStandardsWithTestSuites(
	options: ListAllOptions
): Promise<StandardsWithTestSuites | Error> {
	const result = await listAll(options);
	if (result.isErr) return result.error;
	return result.value;
}
