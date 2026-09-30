// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { ZodError, type ZodType } from 'zod';

import { pb } from '@/pocketbase';

import { nestSuites } from './nest';
import {
	compileCheckListQuery,
	compileSuiteListQuery,
	type CheckListIntent,
	type CompiledListQuery,
	type FilterCompiler,
	type SuiteFacets,
	type SuiteListIntent
} from './query';
import {
	CONFORMANCE_CHECKS_COLLECTION,
	CONFORMANCE_SUITES_COLLECTION,
	conformanceCheckRecordSchema,
	conformanceSuiteRecordSchema,
	type CatalogSurface,
	type ConformanceCheckRecord,
	type ConformanceSuiteRecord
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

/** Await a true-myth Task into Promise<T | Error> for loaders / queryFn. */
export async function awaitTask<T>(task: Task.Task<T, unknown>): Promise<T | Error> {
	const result = await task;
	if (result.isErr) {
		const err = result.error;
		return err instanceof Error ? err : new Error(String(err));
	}
	return result.value;
}

/** Grain-specific compile / collection / schema for {@link listGrainRecords}. */
type GrainListStrategy<TIntent, TRecord> = {
	compile: (intent: TIntent, filter: FilterCompiler) => CompiledListQuery;
	collection: string;
	schema: ZodType<TRecord>;
};

/**
 * Shared catalog list: compile intent → PocketBase getFullList → Zod parse.
 * Grain differences stay in the strategy; callers use typed wrappers.
 */
function listGrainRecords<TIntent extends object, TRecord>(
	options: TIntent & { fetch?: typeof fetch },
	strategy: GrainListStrategy<TIntent, TRecord>
): Task.Task<TRecord[], ClientResponseError | ZodError> {
	const { fetch: fetchFn = fetch, ...intent } = options;
	const compiled = strategy.compile(intent as TIntent, (raw, params) => pb.filter(raw, params));

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
		() => pb.collection(strategy.collection).getFullList(listOptions)
	).andThen((rows) => {
		const parsed = strategy.schema.array().safeParse(rows);
		if (parsed.success) return Task.resolve(parsed.data);
		return Task.reject(parsed.error);
	});
}

/**
 * Shared conformance catalog client: flat checks, suite rows, and nested browse
 * tree — PocketBase adapter over compiled domain intents.
 * Prefer {@link listHubSuites} / {@link listFcafTests} / nest helpers on the
 * public barrel; grain lists stay package-internal (ADR-0010).
 */
export function listChecks(
	options: ListChecksOptions
): Task.Task<ConformanceCheckRecord[], ListChecksError> {
	return listGrainRecords(options, {
		compile: compileCheckListQuery,
		collection: CONFORMANCE_CHECKS_COLLECTION,
		schema: conformanceCheckRecordSchema
	});
}

/**
 * Suite-grain list (package-internal). Public hub Product-axis entry is
 * {@link listHubSuites}; nest compose uses this via {@link listNest}.
 * Default sort intent: wallet→issuer→verifier, then standard, then suite uid.
 */
export function listSuites(
	options: ListSuitesOptions
): Task.Task<ConformanceSuiteRecord[], ListSuitesError> {
	return listGrainRecords(options, {
		compile: compileSuiteListQuery,
		collection: CONFORMANCE_SUITES_COLLECTION,
		schema: conformanceSuiteRecordSchema
	});
}

/**
 * Hub Product-axis suite table use case (ADR-0010). Same options as
 * {@link listSuites}; Catalog surface remains required at the call site (ADR-0008).
 */
export function listHubSuites(
	options: ListSuitesOptions
): Task.Task<ConformanceSuiteRecord[], ListSuitesError> {
	return listSuites(options);
}

// --- Browse tree (nested standards → versions → suites) ---

/** Filesystem-axis nest tree (standards → versions → suites). */
export type NestStandards = Standard[];
export type ListNestError = ClientResponseError | ZodError;

export type ListNestOptions = {
	fetch?: typeof fetch;
	/** Catalog surface — required; no silent default (ADR-0004 / ADR-0008). */
	surface: CatalogSurface;
	facets?: SuiteFacets;
};

/**
 * Package-internal nest compose (listSuites → nestSuites). Not barrel-exported
 * (ADR-0004): client callers use Store.load; SSR / non-hydrating reads use
 * {@link getNestStandards}.
 */
export function listNest(options: ListNestOptions): Task.Task<NestStandards, ListNestError> {
	const { fetch: fetchFn = fetch, surface, facets } = options;

	return listSuites({ fetch: fetchFn, surface, facets }).andThen((records) => {
		const nested = nestSuites(records);
		const res = standardSchema.array().safeParse(nested);
		if (res.success) return Task.resolve(res.data);
		return Task.reject(res.error);
	});
}

/**
 * Sole SSR / non-hydrating Filesystem-axis nest one-shot (ADR-0004). Catalog
 * surface is required at the call site. Does not hydrate Store.
 */
export async function getNestStandards(
	options: ListNestOptions
): Promise<NestStandards | Error> {
	return awaitTask(listNest(options));
}
