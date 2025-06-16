// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getContentBySlug } from '$lib/content';
import tagsIndex from '$lib/content/tags-list.generated.json';
import type { ContentPage } from '$lib/content/types';

export const load = async ({ url }) => {
  const paramTag = url.searchParams.get('search');
  if (!paramTag || !(paramTag in tagsIndex)) {
    return { pages: [] };
  }

  const contentPages = (
    await Promise.all(
      (tagsIndex[paramTag as keyof typeof tagsIndex] as string[]).map(slug =>
        getContentBySlug(slug)
      )
    )
  ).filter(Boolean) as ContentPage[];

  return { pages: contentPages, search: paramTag };
};
