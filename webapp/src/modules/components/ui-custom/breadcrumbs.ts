// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Link } from '@/components/types';
import type { Page } from '@sveltejs/kit';

/* -- Base types -- */

type Param = `[${string}]` | `[[${string}]]`;
type Directory = string;

type BreadcrumbRenamer = (value: string) => string | Promise<string>;

/* -- Main function -- */

export type CalcBreadcrumbsOptions = {
	renamers: Record<Param | Directory, BreadcrumbRenamer>;
	exclude: Array<Param | Directory>;
};

export async function calcBreadcrumbs(
	page: Page,
	options: Partial<CalcBreadcrumbsOptions>
): Promise<Link[]> {
	const { exclude = [], renamers = {} } = options;

	const routeChunks = getPathChunkList(page.url.pathname!);

	const routeChunkDataList = routeChunks
		.map((c) => getRouteChunkData(page, c))
		.filter((c): c is RouteChunkData => Boolean(c))
		.filter((c) => !exclude.includes(c.chunk));

	const breadcrumbDataList = routeChunkDataList
		.map((c) => routeChunkDataToBreadcrumbData(routeChunkDataList, c))
		.map((b) => {
			const renamer: BreadcrumbRenamer | undefined = renamers[b.chunk];
			if (renamer) return renameBreadcrumbData(b, renamer);
			else return b;
		});

	return (await Promise.all(breadcrumbDataList)).map((b) => ({
		title: b.name,
		href: b.url
	}));
}

/* -- Splitting SvelteKit route into chunks, and grouping all the relative data -- */

type RouteBaseChunkData = {
	chunk: Directory;
	isParam: false;
};

type RouteParamChunkData = {
	chunk: Param;
	isParam: true;
	paramValue: string;
};

type RouteChunkData = RouteBaseChunkData | RouteParamChunkData;

function getRouteChunkData(page: Page, routeChunk: string): RouteChunkData | undefined {
	const isParam = stringIsParam(routeChunk);

	if (!isParam) {
		return {
			chunk: routeChunk,
			isParam
		};
	}

	const paramValue = getParamValue(page, routeChunk);

	if (!paramValue) return undefined;
	// This happens if the routeChunk is optional param [[...]]

	return {
		chunk: routeChunk,
		isParam,
		paramValue
	};
}

// Utils

function getPathChunkList(path: string): string[] {
	return path.split('/').filter((chunk) => Boolean(chunk));
}

function stringIsParam(string: string): string is Param {
	return string.startsWith('[') && string.endsWith(']');
}

function getParamName(param: Param): string {
	return param.replaceAll('[', '').replaceAll(']', '');
}

function getParamValue(page: Page, param: Param): string | undefined {
	return page.params[getParamName(param)];
}

/* -- Converting RouteChunkData into useful data for Breadcrumbs -- */

type BreadcrumbData = RouteChunkData & {
	url: string;
	name: string;
};

function routeChunkDataToBreadcrumbData(
	chunkDataList: RouteChunkData[],
	chunkData: RouteChunkData
): BreadcrumbData {
	return {
		...chunkData,
		url: getRouteChunkDataUrl(chunkDataList, chunkData),
		name: getUrlChunk(chunkData)
	};
}

// Utils

function getUrlChunk(routeChunkData: RouteChunkData): string {
	return !routeChunkData.isParam ? routeChunkData.chunk : routeChunkData.paramValue;
}

function getRouteChunkDataUrl(chunkDataList: RouteChunkData[], chunkData: RouteChunkData): string {
	const routeChunkDataIndex = chunkDataList.indexOf(chunkData);
	const urlChunks = chunkDataList.slice(0, routeChunkDataIndex + 1).map(getUrlChunk);
	return `/${urlChunks.join('/')}`;
}

async function renameBreadcrumbData(
	breadcrumbData: BreadcrumbData,
	renamer: BreadcrumbRenamer
): Promise<BreadcrumbData> {
	return {
		...breadcrumbData,
		name: await renamer(breadcrumbData.name)
	};
}
