<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Item from '$lib/components/ui/item/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { Pencil, ChartLine, CalendarClock, ArrowRight } from '@lucide/svelte';
	import { api } from '#lib/api';
	import { research, calendar, articles, siteId, sites, refreshSite, intentName, shortDate } from '#lib/app';

	let error = '', seo: { configured: boolean; connected: boolean; property?: string; totals?: { clicks: number; impressions: number; position: number } } | null = null, seoBusy = false;

	$: r = $research;
	$: cal = $calendar;
	$: domain = $sites.find((s) => s.id === $siteId)?.domain || '';
	$: published = $articles.filter((a) => a.status === 'published').sort((a, b) => (b.published_at || '').localeCompare(a.published_at || ''));
	$: drafts = $articles.filter((a) => a.status !== 'published');
	$: latest = published[0];
	$: upcoming = cal?.upcoming || [];
	$: next = upcoming[0];
	$: draft = drafts[0];
	$: working = r && (r.status === 'queued' || r.status === 'running');
	$: mode = !r?.autopilot ? 'off' : r.auto_publish ? 'auto' : 'draft';

	const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
	const longDate = (s: string) => cap(new Date(s).toLocaleDateString('es-MX', { weekday: 'long', day: 'numeric', month: 'long' }).replace(',', ''));
	const parts = (s: string) => {
		const d = new Date(s);
		return { dow: d.toLocaleDateString('es-MX', { weekday: 'short' }).replace('.', ''), day: d.getDate(), mon: d.toLocaleDateString('es-MX', { month: 'short' }).replace('.', '') };
	};

	// Mini calendario: el mes de la próxima publicación (o el actual), con los días que tienen publicación.
	$: ref = next ? new Date(next.at) : new Date();
	$: year = ref.getFullYear();
	$: month = ref.getMonth();
	$: offset = (new Date(year, month, 1).getDay() + 6) % 7;
	$: days = Array.from({ length: new Date(year, month + 1, 0).getDate() }, (_, i) => i + 1);
	$: marked = (() => {
		const set: number[] = [];
		const add = (at: string) => {
			const d = new Date(at);
			if (d.getFullYear() === year && d.getMonth() === month && !set.includes(d.getDate())) set.push(d.getDate());
		};
		upcoming.forEach((u) => add(u.at));
		(cal?.articles || []).forEach((a) => a.status === 'published' && add(a.at));
		return set;
	})();
	$: monthName = cap(new Date(year, month, 1).toLocaleDateString('es-MX', { month: 'long' }));

	async function loadSeo() {
		if (!$siteId) return;
		try {
			seo = await api(`/sites/${$siteId}/analytics?range=28d`);
		} catch {
			seo = null;
		}
	}
	$: if ($siteId) loadSeo();

	async function connectGoogle() {
		seoBusy = true;
		try {
			if (!seo?.configured) {
				location.href = '/dashboard/analytics';
				return;
			}
			const g = await api(`/sites/${$siteId}/google/start`, { method: 'POST' });
			location.href = g.url;
		} catch (e) {
			error = (e as Error).message;
			seoBusy = false;
		}
	}
	async function toggle(enabled: boolean) {
		error = '';
		try {
			await api(`/sites/${$siteId}/autopilot`, { method: 'PATCH', body: JSON.stringify({ enabled, posts_per_week: r?.posts_per_week || 2 }) });
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		}
	}
	const n = (v: number) => Math.round(v).toLocaleString('es-MX');
</script>

{#if working}
	<Alert.Root role="status"><Spinner /><Alert.Description>Leyendo tu sitio, entendiendo tu negocio y eligiendo los primeros temas… tarda menos de un minuto.</Alert.Description></Alert.Root>
{:else if r?.status === 'failed'}
	<Alert.Root variant="destructive" role="alert"><Alert.Description>{r.error || 'No pude analizar el sitio.'}</Alert.Description></Alert.Root>
{:else if r}
	{#if !r.connected}
		<Alert.Root>
			<Alert.Title>Tu blog ya funciona, pero aún no está en tu sitio.</Alert.Title>
			<Alert.Description>Si abres {r.canonical_url}{r.blog_prefix} hoy verás un 404. <a class="underline" href="/dashboard/conectar">Conéctalo en 2 minutos</a>.</Alert.Description>
		</Alert.Root>
	{/if}
	{#if r.research?.sensitive && !r.auto_publish}
		<Alert.Root>
			<Alert.Title>Sector sensible ({r.research.sensitive_reason || 'salud, legal o finanzas'}).</Alert.Title>
			<Alert.Description>Google pide experiencia y revisión de un profesional en estos temas, así que los artículos quedan en borrador hasta que los apruebes.</Alert.Description>
		</Alert.Root>
	{/if}
	{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}

	<!-- Próxima publicación -->
	<Card.Root>
		<Card.Header>
			<Badge variant="secondary" class="w-fit">{cal?.generating.length ? 'Escribiendo ahora' : 'Próxima publicación'}</Badge>
			<Card.Title class="text-3xl"><h1>{cal?.generating.length ? 'La IA está escribiendo' : next ? longDate(next.at) : 'Sin publicaciones programadas'}</h1></Card.Title>
			<Card.Description>{#if mode === 'off'}El piloto automático está apagado: nada se escribe hasta que lo enciendas o crees un artículo.{:else}La IA escribe, ilustra y {mode === 'auto' ? 'publica' : 'deja en borrador'} en {domain}. Tú solo revisas si quieres.{/if}</Card.Description>
			<Card.Action>
				<label class="flex cursor-pointer items-center gap-3 text-sm">
					<span class="flex flex-col text-right"><b>Piloto automático</b><span class="text-xs text-muted-foreground">{mode === 'auto' ? 'Activo · publica solo' : mode === 'draft' ? 'Activo · deja borradores' : 'Apagado'}</span></span>
					<Switch checked={r.autopilot} onCheckedChange={(v) => toggle(v)} aria-label="Piloto automático" />
				</label>
			</Card.Action>
		</Card.Header>
		<Card.Content>
			{#if cal?.generating.length}
				<Item.Root variant="outline">
					<Item.Media variant="icon"><Spinner /></Item.Media>
					<Item.Content><Item.Title>{cal.generating[0].title}</Item.Title><Item.Description>{r.images_pending ? 'Preparando su imagen…' : 'Redactando el artículo…'}</Item.Description></Item.Content>
				</Item.Root>
			{:else if draft}
				<Item.Root variant="outline">
					<Item.Media variant="icon"><Pencil /></Item.Media>
					<Item.Content><Item.Title>{draft.title}</Item.Title><Item.Description>Borrador listo para revisar</Item.Description></Item.Content>
					<Item.Actions><Button variant="secondary" size="sm" href="/dashboard/articulos?abrir={draft.id}">Ver borrador</Button></Item.Actions>
				</Item.Root>
			{:else if next}
				<Item.Root variant="outline">
					<Item.Media variant="icon"><CalendarClock /></Item.Media>
					<Item.Content><Item.Title>{next.title}</Item.Title><Item.Description>{intentName[next.intent] || next.intent} · palabra clave: {next.keyword}</Item.Description></Item.Content>
					<Item.Actions><Button variant="secondary" size="sm" href="/dashboard/calendario">Ver calendario</Button></Item.Actions>
				</Item.Root>
			{/if}
		</Card.Content>
	</Card.Root>

	<!-- Cifras -->
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<Card.Root><Card.Header><Card.Description>Publicados</Card.Description><Card.Title class="text-3xl">{published.length}</Card.Title></Card.Header><Card.Footer class="text-sm text-muted-foreground">En {domain}</Card.Footer></Card.Root>
		<Card.Root><Card.Header><Card.Description>Por revisar</Card.Description><Card.Title class="text-3xl">{drafts.length}</Card.Title></Card.Header><Card.Footer class="text-sm text-muted-foreground">{drafts.length ? 'Esperan tu revisión' : 'Nada pendiente'}</Card.Footer></Card.Root>
		<Card.Root><Card.Header><Card.Description>Programados</Card.Description><Card.Title class="text-3xl">{upcoming.length}</Card.Title></Card.Header><Card.Footer class="text-sm text-muted-foreground">Próximas semanas</Card.Footer></Card.Root>
		<Card.Root><Card.Header><Card.Description>Por semana</Card.Description><Card.Title class="text-3xl">{r.effective_per_week ?? r.posts_per_week}</Card.Title></Card.Header><Card.Footer class="text-sm text-muted-foreground">{r.ramp_until && r.posts_per_week > 1 ? `Sube a ${r.posts_per_week} el ${shortDate(r.ramp_until)}` : 'Ritmo actual'}</Card.Footer></Card.Root>
	</div>

	<div class="grid items-start gap-4 lg:grid-cols-3">
		<Card.Root class="lg:col-span-2">
			<Card.Header>
				<Card.Title>Cola de publicación</Card.Title>
				<Card.Description>{r.effective_per_week ?? r.posts_per_week} por semana · {upcoming.length} programados</Card.Description>
			</Card.Header>
			<Card.Content>
				<Item.Group>
					{#each upcoming.slice(0, 4) as u}
						{@const d = parts(u.at)}
						<Item.Root variant="outline">
							<Item.Media variant="icon" class="size-12 flex-col gap-0 text-xs"><span class="uppercase">{d.dow}</span><b class="text-base leading-none">{d.day}</b><span class="uppercase">{d.mon}</span></Item.Media>
							<Item.Content><Item.Title>{u.title}</Item.Title><Item.Description class="flex flex-wrap items-center gap-2"><Badge variant="secondary">{intentName[u.intent] || u.intent}</Badge>{u.keyword}</Item.Description></Item.Content>
						</Item.Root>
					{:else}
						<p class="text-sm text-muted-foreground">{r.autopilot ? 'La IA planea los próximos temas en cuanto termina cada artículo.' : 'Enciende el piloto automático para que la IA programe tus artículos.'}</p>
					{/each}
				</Item.Group>
			</Card.Content>
			{#if upcoming.length}<Card.Footer><Button variant="link" href="/dashboard/calendario">Ver los {upcoming.length} en el calendario<ArrowRight /></Button></Card.Footer>{/if}
		</Card.Root>

		<div class="flex flex-col gap-4">
			<Card.Root>
				<Card.Header><Card.Title>{monthName}</Card.Title><Card.Description>{year}</Card.Description></Card.Header>
				<Card.Content>
					<div class="grid grid-cols-7 gap-1 text-center text-sm" data-testid="minical">
						{#each ['L', 'M', 'M', 'J', 'V', 'S', 'D'] as w}<span class="pb-1 text-xs text-muted-foreground">{w}</span>{/each}
						{#each Array(offset) as _}<span></span>{/each}
						{#each days as d}<span class="grid h-8 place-items-center rounded-md {marked.includes(d) ? 'bg-primary font-medium text-primary-foreground' : ''}" data-marked={marked.includes(d)}>{d}</span>{/each}
					</div>
				</Card.Content>
				<Card.Footer class="gap-2 text-sm text-muted-foreground"><span class="size-3 rounded-full bg-primary"></span>Día con publicación</Card.Footer>
			</Card.Root>

			<Card.Root>
				<Card.Header><Card.Title>Rendimiento SEO</Card.Title></Card.Header>
				<Card.Content>
					{#if seo?.connected && seo.property && seo.totals}
						<div class="grid grid-cols-3 gap-2 text-center">
							<div class="flex flex-col"><b class="text-2xl">{n(seo.totals.clicks)}</b><span class="text-xs text-muted-foreground">Clics</span></div>
							<div class="flex flex-col"><b class="text-2xl">{n(seo.totals.impressions)}</b><span class="text-xs text-muted-foreground">Impresiones</span></div>
							<div class="flex flex-col"><b class="text-2xl">{seo.totals.position ? '#' + seo.totals.position.toFixed(1) : '—'}</b><span class="text-xs text-muted-foreground">Posición</span></div>
						</div>
					{:else}
						<div class="flex flex-col items-start gap-3">
							<ChartLine class="size-6 text-muted-foreground" />
							<p class="text-sm text-muted-foreground">Conecta Google Search Console para ver clics, impresiones y posiciones de cada palabra clave.</p>
							<Button size="sm" onclick={connectGoogle} disabled={seoBusy}>{seoBusy ? 'Abriendo…' : 'Conectar Search Console'}</Button>
						</div>
					{/if}
				</Card.Content>
				{#if seo?.connected && seo.property && seo.totals}<Card.Footer><Button variant="link" href="/dashboard/analytics">Ver Analytics<ArrowRight /></Button></Card.Footer>{/if}
			</Card.Root>
		</div>
	</div>

	<!-- Último artículo -->
	{#if latest}
		<Card.Root class="md:flex-row md:items-center" data-testid="latest">
			<div class="p-4 md:w-80 md:shrink-0 md:p-6 md:pr-0">
				{#if latest.has_image}<img class="aspect-video w-full rounded-lg object-cover" src={`/api/articles/${latest.id}/image`} alt="" loading="lazy" />{:else}<div class="grid aspect-video w-full place-items-center rounded-lg bg-muted text-sm text-muted-foreground">{r.images_pending ? 'Creando imagen…' : 'Sin imagen'}</div>{/if}
			</div>
			<div class="flex flex-1 flex-col gap-3">
				<Card.Header>
					<div class="flex items-center gap-2"><Badge>Publicado</Badge><span class="text-sm text-muted-foreground">Último artículo</span></div>
					<Card.Title class="text-xl"><h2>{latest.title}</h2></Card.Title>
					<Card.Description>{latest.summary}</Card.Description>
				</Card.Header>
				<Card.Footer class="gap-2"><Button variant="secondary" size="sm" href="{r.blog_base}/{latest.slug}" target="_blank" rel="noopener">Ver en el sitio</Button><Button variant="outline" size="sm" href="/dashboard/articulos?abrir={latest.id}">Editar</Button></Card.Footer>
			</div>
		</Card.Root>
	{/if}
{/if}
