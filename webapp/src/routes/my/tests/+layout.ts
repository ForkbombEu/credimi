export const load = async ({ parent }) => {
	const { organization } = await parent();

	return { organization };
};
