import type { Handle } from '@sveltejs/kit/hooks';

// El blog público de los clientes va en tema claro; el resto de Faro, en oscuro. Se decide en el servidor porque el blog no lleva JavaScript.
const publicBlog = /^\/(blog|p|x)(\/|$)/;

export const handle: Handle = ({ event, resolve }) => {
	const light = publicBlog.test(event.url.pathname) || event.request.headers.has('x-faro-domain');
	return resolve(event, { transformPageChunk: ({ html }) => html.replace('%theme%', light ? '' : 'dark') });
};
