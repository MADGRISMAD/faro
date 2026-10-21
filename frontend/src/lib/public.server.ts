const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || 'http://localhost:18081';
import { error } from '@sveltejs/kit';
export async function siteFor(request:Request, fetchFn:typeof fetch){
 const domain=request.headers.get('x-faro-domain');
 const target=domain?`/public/domain/${encodeURIComponent(domain)}`:process.env.PUBLIC_SITE_ID?`/public/sites/${process.env.PUBLIC_SITE_ID}`:null;
 if(!target) throw error(404,'Configura un sitio para este blog');
 const r=await fetchFn(`${BACKEND_BASE_URL}${target}`);if(!r.ok) throw error(404,'Sitio no encontrado');return await r.json();
}
export async function publicFetch(path:string,fetchFn:typeof fetch){const r=await fetchFn(`${BACKEND_BASE_URL}${path}`);if(!r.ok) throw error(r.status===404?404:502,r.status===404?'Artículo no encontrado':'No se pudo cargar el blog');return await r.json()}
