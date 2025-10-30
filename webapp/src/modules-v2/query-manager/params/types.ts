// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { Array, Tuple } from 'effect';
import z from 'zod';

import type { CollectionName } from '@/pocketbase/collections-models';

import { ensureArray } from '@/utils/other';

import { DEFAULT_PAGE, DEFAULT_PER_PAGE, type Field } from './utils';

/* Sort */

export type SortParam<C extends CollectionName> = MaybeArray<
	[Field<C>, z.infer<typeof SortOrderSchema>]
>;

export const SortOrderSchema = z.enum(['ASC', 'DESC']);

const SortParamItemSchema = z.codec(z.string(), z.tuple([z.string(), SortOrderSchema]), {
	decode: (value, ctx) => {
		const parts = value.split(DIV);
		if (parts.length !== 2) {
			return codecError(ctx, value, 'Invalid sort param');
		}
		const order = SortOrderSchema.parse(parts[1]);
		return Tuple.make(parts[0], order);
	},
	encode: (value) => `${value[0]}${DIV}${value[1]}`
});

const SortParamSchema = maybeArray(SortParamItemSchema);

/* Filter */

const FilterModeSchema = z.enum(['OR', 'AND']);

export const CompoundFilterSchema = z.codec(
	z.string(),
	z.object({
		id: z.string(),
		// Sometimes we need to update the filter expression from the UI, so we need to keep the id
		expressions: z.array(z.string()),
		mode: FilterModeSchema
	}),
	{
		decode: (value, ctx) => {
			const parts = value.split(DIV);
			if (parts.length !== 3) {
				return codecError(ctx, value, 'Invalid compound filter');
			}
			const id = parts[0];
			const mode = FilterModeSchema.parse(parts[1]);
			const expressions = parts[2].split(SEP);
			return { id, mode, expressions };
		},
		encode: (value) => {
			return [value.id, value.mode, value.expressions.join(SEP)].join(DIV);
		}
	}
);

export type FilterParam = z.infer<typeof FilterParamSchema>;
const FilterParamSchema = maybeArray(z.union([CompoundFilterSchema, z.string()]));
// `compound` must go first because it's more specific

/* Search */

export type SearchParam<C extends CollectionName> = [text: string, fields: MaybeArray<Field<C>>];

const SearchParamSchema = z.codec(z.string(), z.tuple([z.string(), maybeArray(z.string())]), {
	decode: (value, ctx) => {
		const parts = value.split(DIV);
		if (parts.length !== 2) {
			return codecError(ctx, value, 'Invalid search param');
		}
		const text = parts[0];
		const fields = parts[1].split(SEP);
		if (Array.isNonEmptyArray(fields)) {
			return Tuple.make(text, fields);
		} else {
			return codecError(ctx, value, 'Invalid search param');
		}
	},
	encode: ([text, fields]) => {
		return [text, ensureArray(fields).join(SEP)].join(DIV);
	}
});

/* Exclude */

export type ExcludeParam = z.infer<typeof ExcludeParamSchema>;
const ExcludeParamSchema = nonEmptyArray(z.string());

/* Query params */

export type QueryParams<C extends CollectionName> = Partial<{
	page: number;
	perPage: number;
	filter: FilterParam;
	sort: SortParam<C>;
	search: SearchParam<C>;
	excludeIDs: ExcludeParam;
}>;

export const QueryParamsSchema = z
	.object({
		page: z.coerce.number().default(DEFAULT_PAGE),
		perPage: z.coerce.number().default(DEFAULT_PER_PAGE),
		filter: FilterParamSchema,
		sort: SortParamSchema,
		search: SearchParamSchema,
		excludeIDs: ExcludeParamSchema
	})
	.partial();

/* Utils */

const DIV = '+';
const SEP = '--';

type MaybeArray<T> = T | T[];

function nonEmptyArray<Z extends z.ZodTypeAny>(schema: Z) {
	return z.tuple([schema], schema);
}

function maybeArray<Z extends z.ZodTypeAny>(schema: Z) {
	return z.union([schema, nonEmptyArray(schema)]);
}

function codecError<T>(ctx: z.core.ParsePayload<T>, value: T, message: string) {
	ctx.issues.push({
		code: 'custom',
		input: value,
		message: message
	});
	return z.NEVER;
}
