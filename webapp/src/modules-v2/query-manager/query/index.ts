export function build<C extends CollectionName>(query: QueryParams<C>): RecordListOptions {
	const { ...rest } = query;

	const rootParams: QueryParams<C> = rest;
	const urlParams: QueryParams<C> = deserialize(url?.searchParams.toString() ?? '');
	const params = merge(rootParams, urlParams);

	const listOptions: RecordListOptions = {
		page,
		perPage,
		fetch: fetchFn,
		requestKey
	};

	if (expand) listOptions.expand = expand.join(',');
	if (sort) listOptions.sort = buildSortParam(sort);

	const filters: string[] = [];
	if (filter) filters.push(buildFilterParam(filter));
	if (search) filters.push(buildSearchParam(search));
	if (excludeIDs) filters.push(buildExcludeParam(excludeIDs));

	return listOptions;
}
