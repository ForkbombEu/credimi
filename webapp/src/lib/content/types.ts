import { z } from 'zod';

export const pageFrontMatterSchema = z.object({
	date: z.coerce.date(),
	updatedOn: z.coerce.date(),
	title: z.string(),
	description: z.string().optional(),
	tags: z.array(z.string())
});

export type PageFrontMatter = z.infer<typeof pageFrontMatterSchema>;

export type ContentPage = {
	attributes: PageFrontMatter;
	body: string;
	slug: string;
};
