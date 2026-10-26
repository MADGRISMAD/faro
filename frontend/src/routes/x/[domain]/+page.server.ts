import { proxiedSite, publicFetch } from '#lib/public.server';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ fetch, params }) => {
	const site = await proxiedSite(params.domain, fetch);
	const articles = await publicFetch(`/public/sites/${site.id}/articles`, fetch);
	return { site, articles };
};
