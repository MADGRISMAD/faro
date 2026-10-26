import { hostedSite, publicFetch } from '#lib/public.server';
import type { PageServerLoad } from './$types';
export const load: PageServerLoad = async ({ url, fetch, params }) => {
	const site = await hostedSite(params.domain, process.env.PUBLIC_BASE_URL || url.origin, fetch);
	const article = await publicFetch(`/public/sites/${site.id}/articles/${encodeURIComponent(params.slug)}`, fetch);
	return { site, article };
};
