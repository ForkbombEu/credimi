// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

/**
 * Crawler accessibility check for the public SEO pages, robots.txt and sitemap.xml.
 *
 *   bun scripts/check-crawlability.ts [base-url]     (default http://localhost:5100)
 *   CRAWL_RENDERED=0 bun scripts/check-crawlability.ts https://credimi.io
 *
 * Every page is fetched with each crawler user agent and must return the same indexable HTML: a 200
 * text/html response, a title, one H1, a canonical pointing at itself, no noindex, no bot challenge,
 * and real text before JavaScript runs. Changing the user agent does not make a request a verified
 * crawler; it only proves the server does not treat these agents differently. With CRAWL_RENDERED
 * unset, Chromium also renders each page and must show the same H1. Exits 1 on any failure.
 */

import { execFileSync } from 'node:child_process';
import { chromium } from 'playwright';

const base = (process.argv[2] ?? 'http://localhost:5100').replace(/\/$/, '');
const rendered = process.env.CRAWL_RENDERED !== '0';

const PAGES = [
	'/',
	'/pages/eudiw-conformance',
	'/pages/eudi-wallet-testing',
	'/pages/eudi-wallet-interoperability-testing',
	'/pages/openid4vc-conformance',
	'/pages/eidas2-wallet-testing',
	'/pages/fcaf-conformance',
	'/pages/conformance-automation',
	'/pages/why-credimi'
];

const AGENTS: Record<string, string> = {
	browser:
		'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36',
	googlebot: 'Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)',
	bingbot: 'Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)',
	'chatgpt-user':
		'Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; ChatGPT-User/1.0',
	'oai-searchbot':
		'Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; OAI-SearchBot/1.0',
	gptbot: 'Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; GPTBot/1.1',
	claudebot: 'Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; ClaudeBot/1.0',
	perplexitybot:
		'Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; PerplexityBot/1.0'
};

// Markers of Cloudflare and similar interstitials, which a crawler would index instead of the page.
const CHALLENGE = /cf-chl|challenge-platform|Just a moment\.\.\.|Attention Required|captcha/i;
const MIN_WORDS = 150;

const failures: string[] = [];
const fail = (where: string, why: string) => failures.push(`${where}: ${why}`);

function attr(html: string, pattern: RegExp): string | undefined {
	return html.match(pattern)?.[1];
}

function visibleWords(html: string): number {
	return html
		.replace(/<(script|style)[\s\S]*?<\/\1>/gi, ' ')
		.replace(/<[^>]+>/g, ' ')
		.split(/\s+/)
		.filter(Boolean).length;
}

async function get(url: string, agent: string) {
	const response = await fetch(url, { headers: { 'user-agent': agent }, redirect: 'follow' });
	return { response, body: await response.text() };
}

type PageReport = { title?: string; h1?: string; hreflang: Map<string, string> };
const reports = new Map<string, PageReport>();

async function checkPage(path: string) {
	const url = base + path;
	const variants = new Set<string>();
	for (const [name, agent] of Object.entries(AGENTS)) {
		const where = `${path} [${name}]`;
		const { response, body } = await get(url, agent);
		if (response.status !== 200) fail(where, `HTTP ${response.status}`);
		if (response.url !== url) fail(where, `redirected to ${response.url}`);
		if (!response.headers.get('content-type')?.startsWith('text/html'))
			fail(where, `content-type ${response.headers.get('content-type')}`);
		if (CHALLENGE.test(body) || response.headers.has('cf-mitigated'))
			fail(where, 'bot challenge served');

		const title = attr(body, /<title>([^<]+)<\/title>/);
		const canonical = attr(body, /<link rel="canonical" href="([^"]+)"/);
		const robots = attr(body, /<meta name="robots" content="([^"]+)"/);
		const h1s = body.match(/<h1[\s>][\s\S]*?<\/h1>/g) ?? [];
		const words = visibleWords(body);

		if (!title) fail(where, 'no <title>');
		if (!attr(body, /<meta name="description" content="([^"]+)"/))
			fail(where, 'no description');
		if (h1s.length !== 1) fail(where, `${h1s.length} H1 elements`);
		if (!canonical) fail(where, 'no canonical');
		else if (new URL(canonical).pathname !== new URL(url).pathname)
			fail(where, `canonical ${canonical}`);
		if (!robots || /noindex/i.test(robots)) fail(where, `robots "${robots}"`);
		if (!attr(body, /<html[^>]*\slang="([^"]+)"/)) fail(where, 'no html lang');
		if (words < MIN_WORDS) fail(where, `${words} words before JavaScript`);

		const h1 = h1s[0]?.replace(/<[^>]+>/g, '').trim();
		variants.add(`${title}|${h1}|${canonical}`);
		const hreflang = new Map(
			[...body.matchAll(/<link rel="alternate" hreflang="([^"]+)" href="([^"]+)"/g)].map(
				([, lang, href]) => [lang, href]
			)
		);
		reports.set(path, { title, h1, hreflang });
	}
	if (variants.size > 1) fail(path, 'user agents receive different pages');
}

async function checkHreflang() {
	for (const [path, { h1, hreflang }] of reports) {
		if (!hreflang.has('x-default')) fail(path, 'no x-default alternate');
		for (const [lang, href] of hreflang) {
			if (lang === 'x-default') continue;
			// Alternates carry the public origin; fetch them from the server under test.
			const target = new URL(href).pathname;
			const { response, body } = await get(base + target, AGENTS.googlebot);
			const where = `${path} hreflang=${lang}`;
			if (response.redirected) fail(where, `${href} redirects to ${response.url}`);
			if (response.status !== 200) {
				fail(where, `${href} returns ${response.status}`);
				continue;
			}
			// The target must be canonical for itself, list the source back (reciprocal), and be a
			// translation: a page that falls back to English must not be announced as another language.
			const canonical = attr(body, /<link rel="canonical" href="([^"]+)"/);
			if (!canonical || new URL(canonical).pathname !== target)
				fail(where, `${href} is canonicalised to ${canonical}`);
			if (
				!body.includes(
					`hreflang="en" href="https://credimi.io${path === '/' ? '/' : path}"`
				)
			)
				fail(where, `${href} does not link back to ${path}`);
			const targetH1 = body
				.match(/<h1[\s>][\s\S]*?<\/h1>/)?.[0]
				.replace(/<[^>]+>/g, '')
				.trim();
			if (lang !== 'en' && targetH1 === h1) fail(where, `${href} is the untranslated page`);
		}
	}
	const titles = [...reports.values()].map((report) => report.title);
	if (new Set(titles).size !== titles.length) fail('pages', 'duplicate titles');
}

async function checkRobots() {
	const { response, body } = await get(`${base}/robots.txt`, AGENTS.googlebot);
	if (response.status !== 200) fail('/robots.txt', `HTTP ${response.status}`);
	if (!response.headers.get('content-type')?.startsWith('text/plain'))
		fail('/robots.txt', `content-type ${response.headers.get('content-type')}`);
	if (!/^Sitemap: \S+\/sitemap\.xml$/m.test(body)) fail('/robots.txt', 'no Sitemap line');
	for (const path of PAGES)
		if (
			new RegExp(`^Disallow: ${path.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`, 'm').test(body)
		)
			fail('/robots.txt', `disallows ${path}`);
}

async function checkSitemap() {
	const { response, body } = await get(`${base}/sitemap.xml`, AGENTS.googlebot);
	if (response.status !== 200) fail('/sitemap.xml', `HTTP ${response.status}`);
	const type = response.headers.get('content-type') ?? '';
	if (!/^(application|text)\/xml/.test(type)) fail('/sitemap.xml', `content-type ${type}`);
	try {
		execFileSync('xmllint', ['--noout', '-'], {
			input: body,
			stdio: ['pipe', 'ignore', 'pipe']
		});
	} catch (error) {
		fail('/sitemap.xml', `not well-formed XML: ${String(error).slice(0, 200)}`);
	}
	if (!body.includes('xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"'))
		fail('/sitemap.xml', 'missing sitemap namespace');
	const locs = [...body.matchAll(/<loc>([^<]+)<\/loc>/g)].map(([, loc]) => new URL(loc).pathname);
	for (const path of PAGES)
		if (!locs.includes(path)) fail('/sitemap.xml', `does not list ${path}`);
	if (locs.some((path) => /^\/(my|api|_|login|temporal-ui)(\/|$)/.test(path)))
		fail('/sitemap.xml', 'lists a private path');
}

async function checkRendered() {
	const browser = await chromium.launch();
	try {
		const page = await browser.newPage();
		for (const path of PAGES) {
			await page.goto(base + path, { waitUntil: 'networkidle' });
			const h1 = (await page.locator('h1').first().textContent())?.trim();
			if (h1 !== reports.get(path)?.h1)
				fail(`${path} [rendered]`, `H1 "${h1}" differs from HTML`);
		}
	} finally {
		await browser.close();
	}
}

for (const path of PAGES) await checkPage(path);
await checkHreflang();
await checkRobots();
await checkSitemap();
if (rendered) await checkRendered();

const checked = `${PAGES.length} pages × ${Object.keys(AGENTS).length} user agents on ${base}`;
if (failures.length) {
	console.error(`FAIL ${checked}\n${failures.map((line) => `  - ${line}`).join('\n')}`);
	process.exit(1);
}
console.log(`OK ${checked}, robots.txt, sitemap.xml${rendered ? ', rendered H1' : ''}`);
