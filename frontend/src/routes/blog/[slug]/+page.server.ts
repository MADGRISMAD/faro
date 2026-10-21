import {siteFor,publicFetch} from '#lib/public.server';import type {PageServerLoad} from './$types';
export const load:PageServerLoad=async({request,fetch,params})=>{const site=await siteFor(request,fetch);const article=await publicFetch(`/public/sites/${site.id}/articles/${encodeURIComponent(params.slug)}`,fetch);return {site,article}};
