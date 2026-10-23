import { z } from 'zod/v3';

export const pageFrontMatterSchema = z.object({
	date: z.coerce.date(),
	updatedOn: z.coerce.date(),
	title: z.string(),
	description: z.string().optional(),
	/** Hero text of the page; `description` stays the search snippet. Defaults to `description`. */
	lead: z.string().optional(),
	tags: z.array(z.string()),
	socialCard: z.string().optional(),
	/** What the social card shows, for og:image:alt. */
	socialCardAlt: z.string().optional(),
	/** Cards on the right of the hero; `role` picks the Hub entity icon and colors. */
	heroCards: z
		.array(
			z.object({
				role: z.enum([
					'credential_issuers',
					'wallets',
					'verifiers',
					'pipelines',
					'conformance_checks'
				]),
				title: z.string(),
				detail: z.string()
			})
		)
		.optional(),
	/** Key figures shown at the bottom of the hero. */
	facts: z.array(z.object({ term: z.string(), value: z.string() })).optional(),
	/** Closing call to action, shown as an indigo band before "Read next". */
	cta: z
		.object({ title: z.string(), text: z.string(), href: z.string(), label: z.string() })
		.optional()
});

export type PageFrontMatter = z.infer<typeof pageFrontMatterSchema>;

export type ContentPage = {
	attributes: PageFrontMatter;
	body: string;
	slug: string;
	/** H2 headings, in order, with the id the body gives them. */
	headings: { id: string; text: string }[];
	/** Questions under the "FAQ" heading, as plain text, for FAQPage structured data. */
	faq: { question: string; answer: string }[];
};
