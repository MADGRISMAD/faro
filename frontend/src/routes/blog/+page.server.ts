import {siteFor,publicFetch} from '#lib/public.server';
import type {PageServerLoad} from './$types';
export const load:PageServerLoad=async({request,fetch})=>{const site=await siteFor(request,fetch);const articles=await publicFetch(`/public/sites/${site.id}/articles`,fetch);return {site,articles}};
