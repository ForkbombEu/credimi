// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { HubItem } from '$lib/hub';

import { getNestStandards } from '$lib/conformance/client';
import { getContentBySlug } from '$lib/content';
import tagsIndex from '$lib/content/tags-list.generated.json';
import { entities } from '$lib/global/entities';
import { getSuitePageUrl, getStandardCheckUrlFromPath } from '$lib/hub/conformance/urls';

import { pb } from '@/pocketbase';

import { plainText, staticPages } from './index';

export type PublicPageSection =
	| 'core'
	| 'guides'
	| 'hub'
	| 'conformance-suites'
	| 'conformance-checks'
	| 'news'
	| 'organizations';

/** One indexable URL, as listed in sitemap.xml and llms.txt. */
export type PublicPage = {
	/** Base-locale path; locale variants are derived from it. */
	path: string;
	title: string;
	/** Plain text, untruncated. */
	description: string;
	section: PublicPageSection;
	lastmod?: string;
	/** Hub entity label, e.g. "Wallet". */
	kind?: string;
};

type Fetch = typeof fetch;

/** Every public page anonymous visitors can see: anything PocketBase hides from them stays out. */
export async function listPublicPages(fetch: Fetch): Promise<PublicPage[]> {
	const sources = await Promise.all([
		contentPages(fetch),
		hubPages(fetch),
		conformancePages(fetch),
		newsPages(fetch),
		organizationPages(fetch)
	]);

	const core: PublicPage[] = Object.entries(staticPages).map(([path, seo]) => ({
		path,
		title: seo.title,
		description: seo.description,
		section: 'core'
	}));

	return [...core, ...sources.flat()];
}

/** A failing source fails the whole listing: crawlers keep the last good sitemap after an error,
 * but replace it with a truncated one. */
async function listSource(source: string, run: () => Promise<PublicPage[]>): Promise<PublicPage[]> {
	try {
		return await run();
	} catch (error) {
		throw new Error(`could not list ${source} pages`, { cause: error });
	}
}

function contentPages(fetch: Fetch) {
	return listSource('content', async () => {
		const slugs = [...new Set(Object.values(tagsIndex).flat())];
		const pages = await Promise.all(slugs.map((slug) => getContentBySlug(slug, fetch)));
		return pages
			.filter((page) => page !== undefined)
			.map((page) => ({
				path: `/pages/${page.slug}`,
				title: page.attributes.title,
				description: page.attributes.description ?? '',
				lastmod: page.attributes.updatedOn.toISOString(),
				section: 'guides' as const
			}));
	});
}

function hubPages(fetch: Fetch) {
	return listSource('hub', async () => {
		const items = await pb.collection('hub_items').getFullList<HubItem>({
			fetch,
			// Custom checks open a private run page, not a public Hub page.
			filter: "type != 'custom_checks'",
			sort: 'type,name'
		});
		return items.map((item) => {
			const kind = entities[item.type].labels.singular;
			return {
				path: `/hub/${item.path}`,
				title: item.name ?? item.canonified_name ?? kind,
				description: plainText(item.description, 10_000),
				lastmod: item.updated ? new Date(item.updated).toISOString() : undefined,
				section: 'hub' as const,
				kind
			};
		});
	});
}

function conformancePages(fetch: Fetch) {
	return listSource('conformance', async () => {
		const standards = await getNestStandards({ fetch, surface: 'pipeline' });
		if (standards instanceof Error) throw standards;

		return standards.flatMap((standard) =>
			standard.versions.flatMap((version) =>
				version.suites.flatMap((suite): PublicPage[] => [
					{
						path: getSuitePageUrl(standard.uid, version.uid, suite.uid),
						title: `${suite.name} ${version.name}: ${standard.name} conformance checks`,
						description: plainText(suite.description, 10_000),
						section: 'conformance-suites'
					},
					...suite.members.map((member) => ({
						path: getStandardCheckUrlFromPath(member.path),
						title: `${member.path.split('/').at(-1)} (${standard.name} ${version.name})`,
						description: `${member.title} ${standard.name} ${version.name} conformance check from ${suite.name}.`,
						section: 'conformance-checks' as const
					}))
				])
			)
		);
	});
}

function newsPages(fetch: Fetch) {
	return listSource('news', async () => {
		const news = await pb.collection('news').getFullList({ fetch, sort: '-created' });
		return news.map((item) => ({
			path: `/news/${item.id}`,
			title: item.title ?? 'News',
			description: plainText(item.summary, 10_000),
			lastmod: new Date(item.updated).toISOString(),
			section: 'news' as const
		}));
	});
}

function organizationPages(fetch: Fetch) {
	return listSource('organizations', async () => {
		const organizations = await pb
			.collection('organizations')
			.getFullList({ fetch, filter: 'published = true', sort: 'name' });
		return organizations
			.filter((organization) => organization.canonified_name)
			.map((organization) => ({
				path: `/organizations/${organization.canonified_name}`,
				title: organization.name,
				description: plainText(organization.description, 10_000),
				lastmod: new Date(organization.updated).toISOString(),
				section: 'organizations' as const
			}));
	});
}
