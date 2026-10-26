const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || 'http://localhost:18081';
import { error } from '@sveltejs/kit';
export async function siteFor(request:Request, fetchFn:typeof fetch){
 const domain=request.headers.get('x-faro-domain');
 const target=domain?`/public/domain/${encodeURIComponent(domain)}`:process.env.PUBLIC_SITE_ID?`/public/sites/${process.env.PUBLIC_SITE_ID}`:null;
 if(!target) throw error(404,'Configura un sitio para este blog');
 const r=await fetchFn(`${BACKEND_BASE_URL}${target}`);if(!r.ok) throw error(404,'Sitio no encontrado');return await r.json();
}
export async function publicFetch(path:string,fetchFn:typeof fetch){const r=await fetchFn(`${BACKEND_BASE_URL}${path}`);if(!r.ok) throw error(r.status===404?404:502,r.status===404?'Artículo no encontrado':'No se pudo cargar el blog');return await r.json()}

// Blog alojado en Faro (/p/<dominio>): funciona sin tocar el sitio del cliente. Los enlaces y el canonical apuntan a esta misma dirección.
export async function hostedSite(domain:string,origin:string,fetchFn:typeof fetch){
 const r=await fetchFn(`${BACKEND_BASE_URL}/public/domain/${encodeURIComponent(domain)}`);if(!r.ok) throw error(404,'Blog no encontrado');
 const site=await r.json();return {...site,canonical_url:origin,blog_prefix:`/p/${domain}`};
}

// Modo proxy sin cabeceras (Vercel, Netlify…): el dominio llega en la ruta (/x/<dominio>) y el blog se ve como en el dominio del cliente:
// enlaces con su prefijo (/blog) y canonical del cliente. Es lo mismo que el modo por cabecera x-faro-domain.
export async function proxiedSite(domain:string,fetchFn:typeof fetch){
 const r=await fetchFn(`${BACKEND_BASE_URL}/public/domain/${encodeURIComponent(domain)}`);if(!r.ok) throw error(404,'Blog no encontrado');
 return await r.json();
}
