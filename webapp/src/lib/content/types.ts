import { z } from 'zod/v3';

export const pageFrontMatterSchema = z.object({
	date: z.coerce.date(),
	updatedOn: z.coerce.date(),
	title: z.string(),
	description: z.string().optional(),
	tags: z.array(z.string()),
	socialCard: z.string().optional(),
	/** What the social card shows, for og:image:alt. */
	socialCardAlt: z.string().optional()
});

export type PageFrontMatter = z.infer<typeof pageFrontMatterSchema>;

export type ContentPage = {
	attributes: PageFrontMatter;
	body: string;
	slug: string;
};
