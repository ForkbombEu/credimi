// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Public pages render on the server so search engines and AI crawlers that do not run
// JavaScript get the content and <head> tags. The rest of the app stays client-only (root layout).
export const ssr = true;
