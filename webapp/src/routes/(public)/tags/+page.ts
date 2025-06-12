// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
import fm from 'front-matter';
import { contentLoaders, type PageFrontMatter, type PageWithBody } from '$lib/content';
import tagsIndex from '$lib/content/tags-list.generated.json';

export const load = async () => {
  const allKeys = Array.from(
    new Set(Object.values(tagsIndex).flat())
  ) as string[];
  const pages = (
    await Promise.all(
      allKeys.map(async (key) => {
        const loader = contentLoaders[key as keyof typeof contentLoaders]!;
        try {
          const raw = await loader();
          const { attributes, body } = fm<PageFrontMatter>(raw);
          return {
            slug: key.match(/content\/[^/]+\/([^/]+)\//)?.[1] ?? '',
            title: attributes.title,
            description: attributes.description,
            date: attributes.date,
            tags: attributes.tags,
            updatedOn: attributes.updatedOn,
            body
          } as PageWithBody;
        } catch (e) {
          console.error('Error loading', key, e);
          return null;
        }
      })
    )
  ).filter((p) => p !== null);

  return { pages };
};
