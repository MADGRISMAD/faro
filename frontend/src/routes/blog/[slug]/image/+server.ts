import { siteFor } from '#lib/public.server';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || 'http://localhost:18081';

// Portada del artículo: se sirve desde el mismo dominio del blog para que Google la asocie al sitio del cliente.
export const GET: RequestHandler = async ({ request, fetch, params }) => {
	const site = await siteFor(request, fetch);
	const r = await fetch(`${BACKEND_BASE_URL}/public/sites/${site.id}/articles/${encodeURIComponent(params.slug)}/image`);
	if (!r.ok) throw error(404, 'Sin imagen');
	return new Response(r.body, { headers: { 'content-type': r.headers.get('content-type') || 'image/jpeg', 'cache-control': 'public, max-age=86400' } });
};
