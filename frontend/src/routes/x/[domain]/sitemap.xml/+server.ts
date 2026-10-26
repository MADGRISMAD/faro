import { proxiedSite, publicFetch } from '#lib/public.server';
import { canonical } from '#lib/seo';
import type { RequestHandler } from './$types';
export const GET: RequestHandler = async ({ fetch, params }) => {
	const site = await proxiedSite(params.domain, fetch);
	const articles = await publicFetch(`/public/sites/${site.id}/articles`, fetch);
	const urls = [canonical(site), ...articles.map((a: { slug: string }) => canonical(site, a.slug))];
	const esc = (u: string) => u.replace(/&/g, '&amp;').replace(/</g, '&lt;');
	const xml = `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls.map((u) => `<url><loc>${esc(u)}</loc></url>`).join('')}</urlset>`;
	return new Response(xml, { headers: { 'content-type': 'application/xml; charset=utf-8' } });
};
