<script lang="ts">
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

<div class="view wide home">
	{#if working}
		<div class="panel" role="status"><span class="spin"></span>Leyendo tu sitio, entendiendo tu negocio y eligiendo los primeros temas… tarda menos de un minuto.</div>
	{:else if r?.status === 'failed'}
		<div class="panel"><p class="error" role="alert">{r.error || 'No pude analizar el sitio.'}</p></div>
	{:else if r}
		{#if !r.connected}
			<a class="panel notice" href="/dashboard/conectar"><b>Tu blog ya funciona, pero aún no está en tu sitio.</b> <span class="muted">Si abres {r.canonical_url}{r.blog_prefix} hoy verás un 404. Conéctalo en 2 minutos →</span></a>
		{/if}
		{#if r.research?.sensitive && !r.auto_publish}
			<div class="panel notice"><b>Sector sensible ({r.research.sensitive_reason || 'salud, legal o finanzas'}).</b> <span class="muted">Google pide experiencia y revisión de un profesional en estos temas, así que los artículos quedan en borrador hasta que los apruebes.</span></div>
		{/if}
		{#if error}<p class="error" role="alert">{error}</p>{/if}

		<!-- Próxima publicación -->
		<section class="panel nextpub">
			<div class="nextpub-top">
				<div>
					<p class="section-eyebrow">{cal?.generating.length ? 'Escribiendo ahora' : 'Próxima publicación'}</p>
					<h1>{cal?.generating.length ? 'La IA está escribiendo' : next ? longDate(next.at) : 'Sin publicaciones programadas'}</h1>
					<p class="muted lead">{#if mode === 'off'}El piloto automático está apagado: nada se escribe hasta que lo enciendas o crees un artículo.{:else}La IA escribe, ilustra y {mode === 'auto' ? 'publica' : 'deja en borrador'} en {domain}. Tú solo revisas si quieres.{/if}</p>
				</div>
				<label class="auto"><span><b>Piloto automático</b><small>{mode === 'auto' ? 'Activo · publica solo' : mode === 'draft' ? 'Activo · deja borradores' : 'Apagado'}</small></span>
					<span class="switch"><input type="checkbox" checked={r.autopilot} on:change={(e) => toggle(e.currentTarget.checked)} aria-label="Piloto automático" /></span></label>
			</div>
			{#if cal?.generating.length}
				<div class="well item"><span class="icon"><span class="spin" style="margin:0"></span></span><div class="grow"><b>{cal.generating[0].title}</b><span class="muted small">{r.images_pending ? 'Preparando su imagen…' : 'Redactando el artículo…'}</span></div></div>
			{:else if draft}
				<div class="well item"><span class="icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20h4L19 9l-4-4L4 16v4zM14 6l4 4" /></svg></span><div class="grow"><b>{draft.title}</b><span class="muted small">Borrador listo para revisar</span></div><a class="btn secondary small" href="/dashboard/articulos?abrir={draft.id}">Ver borrador</a></div>
			{:else if next}
				<div class="well item"><span class="icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20h4L19 9l-4-4L4 16v4zM14 6l4 4" /></svg></span><div class="grow"><b>{next.title}</b><span class="muted small">{intentName[next.intent] || next.intent} · palabra clave: {next.keyword}</span></div><a class="btn secondary small" href="/dashboard/calendario">Ver calendario</a></div>
			{/if}
		</section>

		<!-- Cifras -->
		<div class="stats">
			<div class="stat"><span class="num">{published.length}</span><div><b>Publicados</b><span class="muted small">En {domain}</span></div></div>
			<div class="stat"><span class="num">{drafts.length}</span><div><b>Por revisar</b><span class="muted small">{drafts.length ? 'Esperan tu revisión' : 'Nada pendiente'}</span></div></div>
			<div class="stat"><span class="num">{upcoming.length}</span><div><b>Programados</b><span class="muted small">Próximas semanas</span></div></div>
			<div class="stat"><span class="num">{r.effective_per_week ?? r.posts_per_week}</span><div><b>Por semana</b><span class="muted small">{r.ramp_until && r.posts_per_week > 1 ? `Sube a ${r.posts_per_week} el ${shortDate(r.ramp_until)}` : 'Ritmo actual'}</span></div></div>
		</div>

		<div class="cols">
			<section class="panel queue">
				<div class="row"><h2>Cola de publicación</h2><span class="muted small">{r.effective_per_week ?? r.posts_per_week} por semana · {upcoming.length} programados</span></div>
				{#each upcoming.slice(0, 4) as u}
					{@const d = parts(u.at)}
					<div class="well qitem">
						<span class="datetile"><small>{d.dow}</small><b>{d.day}</b><small class="mon">{d.mon}</small></span>
						<div class="grow"><b>{u.title}</b><span class="meta"><span class="chip">{intentName[u.intent] || u.intent}</span> <span class="muted small">{u.keyword}</span></span></div>
					</div>
				{:else}
					<p class="muted">{r.autopilot ? 'La IA planea los próximos temas en cuanto termina cada artículo.' : 'Enciende el piloto automático para que la IA programe tus artículos.'}</p>
				{/each}
				{#if upcoming.length}<p class="more"><a href="/dashboard/calendario">Ver los {upcoming.length} en el calendario →</a></p>{/if}
			</section>

			<aside class="side-col">
				<section class="panel minical">
					<div class="row"><h2>{monthName}</h2><span class="muted small">{year}</span></div>
					<div class="mc">
						{#each ['L', 'M', 'M', 'J', 'V', 'S', 'D'] as w}<span class="dow">{w}</span>{/each}
						{#each Array(offset) as _}<span></span>{/each}
						{#each days as d}<span class="day" class:on={marked.includes(d)}>{d}</span>{/each}
					</div>
					<p class="legend"><i></i> Día con publicación</p>
				</section>

				<section class="panel seo">
					<h2>Rendimiento SEO</h2>
					{#if seo?.connected && seo.property && seo.totals}
						<div class="well seo-stats">
							<div><b>{n(seo.totals.clicks)}</b><span class="muted small">Clics</span></div>
							<div><b>{n(seo.totals.impressions)}</b><span class="muted small">Impresiones</span></div>
							<div><b>{seo.totals.position ? '#' + seo.totals.position.toFixed(1) : '—'}</b><span class="muted small">Posición</span></div>
						</div>
						<p class="more"><a href="/dashboard/analytics">Ver Analytics →</a></p>
					{:else}
						<div class="well seo-empty">
							<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 4v16h16M8 15l3-4 3 3 5-7" /></svg>
							<p class="muted" style="margin:0">Conecta Google Search Console para ver clics, impresiones y posiciones de cada palabra clave.</p>
							<button class="btn small" on:click={connectGoogle} disabled={seoBusy}>{seoBusy ? 'Abriendo…' : 'Conectar Search Console'}</button>
						</div>
					{/if}
				</section>
			</aside>
		</div>

		<!-- Último artículo -->
		{#if latest}
			<section class="panel latest">
				{#if latest.has_image}<img class="thumb" src={`/api/articles/${latest.id}/image`} alt="" loading="lazy" />{:else}<div class="thumb empty">{r.images_pending ? 'Creando imagen…' : 'Sin imagen'}</div>{/if}
				<div>
					<p><span class="chip published">Publicado</span> <span class="muted small">Último artículo</span></p>
					<h2>{latest.title}</h2>
					<p class="muted">{latest.summary}</p>
					<div class="actions"><a class="btn secondary small" href="{r.blog_base}/{latest.slug}" target="_blank" rel="noopener">Ver en el sitio</a><a class="btn ghost small" href="/dashboard/articulos?abrir={latest.id}">Editar</a></div>
				</div>
			</section>
		{/if}
	{/if}
</div>

<style>
	.home{display:grid;gap:22px;align-content:start}
	.notice{display:block;text-decoration:none;color:inherit;box-shadow:var(--out),0 0 0 3px var(--accent-l)}
	.nextpub{padding:26px 30px;display:block}
	.nextpub-top{display:flex;justify-content:space-between;gap:20px;flex-wrap:wrap;align-items:flex-start}
	.nextpub-top>div:first-child{flex:1 1 360px;min-width:0}
	.section-eyebrow{margin:0;font-size:.72rem;font-weight:800;letter-spacing:.14em;text-transform:uppercase;color:var(--accent-d)}
	.nextpub h1{margin:.2rem 0 0;font-size:clamp(2rem,4.4vw,3.2rem);line-height:1.05}
	.lead{margin:.7rem 0 0;max-width:46rem}
	.auto{flex:none;display:flex;align-items:center;gap:14px;padding:10px 14px 10px 18px;border-radius:22px;background:var(--well);box-shadow:var(--in-s);cursor:pointer}
	.auto span:first-child{display:grid;font-size:.85rem}.auto small{color:var(--mut);font-size:.72rem}
	.switch input{appearance:none;width:52px;height:30px;border-radius:99px;background:var(--surface);box-shadow:var(--in-s);position:relative;padding:0;cursor:pointer}
	.well{border-radius:24px;background:var(--well);box-shadow:var(--in)}
	.item{display:flex;flex-wrap:wrap;align-items:center;gap:12px 16px;margin-top:22px;padding:14px 18px}
	.icon{display:grid;place-items:center;width:46px;height:46px;border-radius:15px;background:var(--accent-xl);box-shadow:var(--out-s);flex:none}
	.icon svg,.seo-empty svg{width:22px;height:22px;fill:none;stroke:var(--accent-d);stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}
	.grow{flex:1 1 220px;min-width:0;display:grid;gap:2px}.grow b{overflow-wrap:anywhere}
	.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:22px}
	.stat{display:flex;align-items:center;gap:16px;padding:18px 22px;border-radius:28px;background:var(--surface);box-shadow:var(--out)}
	.stat>div{display:grid}.stat b{font-size:.98rem}
	.num{display:grid;place-items:center;min-width:60px;height:60px;border-radius:20px;background:var(--well);box-shadow:var(--in-s);font-weight:900;font-size:2rem;color:var(--accent-d);padding:0 8px}
	.cols{display:grid;grid-template-columns:minmax(0,1.7fr) minmax(0,1fr);gap:22px;align-items:start}
	.queue h2,.minical h2,.seo h2,.latest h2{margin:0;font-size:1.7rem}
	.qitem{display:flex;align-items:center;gap:16px;margin-top:14px;padding:12px 16px}
	.datetile{display:grid;justify-items:center;width:62px;padding:8px 0;border-radius:18px;background:var(--surface);box-shadow:var(--out-s);flex:none;line-height:1}
	.datetile small{font-size:.62rem;font-weight:800;letter-spacing:.12em;text-transform:uppercase;color:var(--mut)}.datetile .mon{color:var(--accent-d)}
	.datetile b{font-weight:900;font-size:1.7rem;margin:3px 0}
	.meta{display:flex;gap:10px;align-items:center;flex-wrap:wrap;margin-top:4px}
	.more{margin:18px 0 0;text-align:center;font-weight:800}.more a{color:var(--accent-d);text-decoration:none}
	.side-col{display:grid;gap:22px}
	.mc{display:grid;grid-template-columns:repeat(7,1fr);gap:6px;margin-top:16px;text-align:center;font-size:.9rem}
	.mc .dow{font-size:.72rem;font-weight:800;color:var(--mut);padding-bottom:4px}
	.mc .day{display:grid;place-items:center;height:34px;border-radius:12px}
	.mc .day.on{background:var(--accent-d);color:#fff;font-weight:800;box-shadow:var(--puff)}
	.legend{display:flex;align-items:center;gap:8px;margin:14px 0 0;font-size:.8rem;color:var(--mut)}.legend i{width:11px;height:11px;border-radius:50%;background:var(--accent-d)}
	.seo-empty{display:grid;gap:12px;margin-top:14px;padding:18px;justify-items:start}
	.seo-stats{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin-top:14px;padding:16px;text-align:center}
	.seo-stats div{display:grid}.seo-stats b{font-weight:900;font-size:1.6rem;color:var(--accent-d)}
	.latest{display:grid;grid-template-columns:minmax(0,340px) 1fr;gap:26px;align-items:center}
	.latest .thumb{aspect-ratio:16/9}
	.latest p{margin:.4rem 0}.latest h2{margin:.5rem 0}
	.actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:14px}
	@media(max-width:1000px){.cols{grid-template-columns:1fr}.latest{grid-template-columns:1fr}}
	@media(max-width:560px){.nextpub{padding:20px}.item .btn{margin-left:62px}.auto{width:100%;justify-content:space-between}.qitem{padding:10px 12px;gap:12px}.stat{padding:14px 16px}}
</style>
