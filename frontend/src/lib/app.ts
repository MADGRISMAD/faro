import { writable, get } from 'svelte/store';
import { api } from './api';

export type Site = { id: string; name: string; domain: string; canonical_url: string; blog_prefix: string; brand_description: string; tone: string };
export type Topic = { title: string; keyword: string; intent: string; why: string };
export type Profile = { name: string; business: string; audience: string; tone: string; location: string; offerings: string[]; topics: Topic[]; sensitive: boolean; sensitive_reason: string; platform: string };
export type Research = {
	status: string; error: string; autopilot: boolean; auto_publish: boolean; posts_per_week: number; effective_per_week: number; ramp_until?: string;
	articles_pending: number; articles_done: number; images_pending: number;
	images_enabled: boolean; image_style: string; image_notes: string; avoid_topics: string; research?: Profile;
	brand_confirmed: boolean; connected: boolean; blog_base: string; hosted_url: string; domain: string; canonical_url: string; blog_prefix: string;
};
export type Article = {
	id: string; site_id: string; title: string; slug: string; meta_description: string; summary: string; status: string; has_image: boolean; published_at: string | null;
	// detalle: lo que la revisión automática pide confirmar; lista: cuántas notas tiene
	review_notes?: string[] | number;
	content: { intro: string; sections: { heading: string; level?: number; paragraphs: string[]; bullets: string[] }[]; faq: { question: string; answer: string }[] };
};
export type Calendar = {
	articles: { id: string; title: string; status: string; has_image: boolean; at: string }[];
	generating: { title: string; at: string }[];
	upcoming: { title: string; keyword: string; intent: string; at: string }[];
};

export const user = writable<any>(null);
export const ready = writable(false);
export const sites = writable<Site[]>([]);
export const siteId = writable('');
export const research = writable<Research | null>(null);
export const articles = writable<Article[]>([]);
export const calendar = writable<Calendar | null>(null);
export const addingSite = writable(false);

let timer: ReturnType<typeof setTimeout> | undefined;

export async function boot() {
	try {
		user.set(await api('/auth/me'));
		await loadSites();
	} catch {
		user.set(null);
	} finally {
		ready.set(true);
	}
}

export async function loadSites() {
	const list: Site[] = await api('/sites');
	sites.set(list);
	if (!list.find((s) => s.id === get(siteId))) siteId.set(list[0]?.id || '');
	await refreshSite();
}

export function selectSite(id: string) {
	siteId.set(id);
	addingSite.set(false);
	refreshSite();
}

// Recarga todo lo del sitio activo y, mientras la IA esté trabajando, vuelve a preguntar sola cada 3 s.
export async function refreshSite() {
	if (timer) clearTimeout(timer);
	const id = get(siteId);
	if (!id) {
		research.set(null);
		calendar.set(null);
		return;
	}
	try {
		const [r, a, c] = await Promise.all([api(`/sites/${id}/research`), api('/articles'), api(`/sites/${id}/calendar`)]);
		if (get(siteId) !== id) return;
		research.set(r);
		articles.set(a.filter((x: Article) => x.site_id === id)); // el panel muestra solo el sitio activo
		calendar.set(c);
		// mientras la IA trabaja, cada 3 s; si no, cada minuto (el autopiloto escribe solo y las portadas llegan después del artículo)
		const busy = r.status === 'queued' || r.status === 'running' || r.articles_pending > 0 || r.images_pending > 0;
		timer = setTimeout(refreshSite, busy ? 3000 : 60000);
	} catch {
		/* sin conexión: se reintenta al siguiente evento */
	}
}

// fecha corta en español: «9 de noviembre»
export const shortDate = (d: string) => new Date(d).toLocaleDateString('es-MX', { day: 'numeric', month: 'long' });

export function stopPolling() {
	if (timer) clearTimeout(timer);
}

export const statusName: Record<string, string> = { draft: 'Borrador', approved: 'Aprobado', published: 'Publicado' };
export const intentName: Record<string, string> = { informational: 'Informativo', comparison: 'Comparación', purchase: 'Compra', local: 'Local' };
export const imageStyles: { id: string; name: string; hint: string }[] = [
	{ id: 'clean', name: 'Limpio y minimalista', hint: 'Ilustración plana, como de revista' },
	{ id: 'photo', name: 'Fotorrealista', hint: 'Fotografía editorial con luz natural' },
	{ id: 'watercolor', name: 'Acuarela', hint: 'Pintura suave sobre papel' },
	{ id: '3d', name: 'Render 3D', hint: 'Figuras tipo arcilla o juguete' },
	{ id: 'vintage', name: 'Vintage', hint: 'Cartel retro con trama de puntos' },
	{ id: 'abstract', name: 'Abstracto', hint: 'Formas y colores intensos' }
];
