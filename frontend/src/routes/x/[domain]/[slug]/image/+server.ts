import { proxiedSite } from '#lib/public.server';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || 'http://localhost:18081';
export const GET: RequestHandler = async ({ fetch, params }) => {
	const site = await proxiedSite(params.domain, fetch);
	const r = await fetch(`${BACKEND_BASE_URL}/public/sites/${site.id}/articles/${encodeURIComponent(params.slug)}/image`);
	if (!r.ok) throw error(404, 'Sin imagen');
	return new Response(r.body, { headers: { 'content-type': r.headers.get('content-type') || 'image/jpeg', 'cache-control': 'public, max-age=86400' } });
};
