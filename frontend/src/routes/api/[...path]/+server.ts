const BACKEND_BASE_URL = process.env.BACKEND_BASE_URL || 'http://localhost:18081';
import type { RequestHandler } from './$types';
const handler: RequestHandler = async ({ request, params, url }) => {
 const headers = new Headers(request.headers); headers.delete('host'); headers.delete('content-length');
 headers.set('x-forwarded-proto', url.protocol.slice(0,-1));
 const response = await fetch(`${BACKEND_BASE_URL}/${params.path}${url.search}`, { method: request.method, headers, body: ['GET','HEAD'].includes(request.method) ? undefined : await request.arrayBuffer(), redirect:'manual' });
 const out = new Headers(response.headers); out.delete('content-encoding'); out.delete('content-length');
 return new Response(response.body,{status:response.status,headers:out});
};
export const GET=handler; export const POST=handler; export const PATCH=handler; export const DELETE=handler;
