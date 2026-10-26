<script lang="ts">
	import { api } from '#lib/api';
	import { research, calendar, articles, siteId, user, refreshSite, intentName } from '#lib/app';

	let error = '';
	$: r = $research;
	$: cal = $calendar;
	$: published = $articles.filter((a) => a.status === 'published');
	$: drafts = $articles.filter((a) => a.status !== 'published');
	$: recent = [...$articles].sort((a, b) => (b.published_at || '').localeCompare(a.published_at || '')).slice(0, 3);
	$: left = $user ? ($user.budget_microusd - $user.spent_microusd - $user.reserved_microusd) / 1e6 : 0;
	$: working = r && (r.status === 'queued' || r.status === 'running');

	const day = (s: string) => new Date(s).toLocaleDateString('es-MX', { weekday: 'long', day: 'numeric', month: 'short' });
	async function toggle(enabled: boolean) {
		error = '';
		try {
			await api(`/sites/${$siteId}/autopilot`, { method: 'PATCH', body: JSON.stringify({ enabled, posts_per_week: r?.posts_per_week || 2 }) });
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		}
	}
</script>

<div class="view">
	<h1>Inicio</h1>
	<p class="sub">La IA se encarga de tu contenido. Aquí ves cómo va.</p>

	{#if working}
		<div class="panel" role="status"><span class="spin"></span>Leyendo tu sitio, entendiendo tu negocio y eligiendo los primeros temas… tarda menos de un minuto.</div>
	{:else if r?.status === 'failed'}
		<div class="panel"><p class="error" role="alert">{r.error || 'No pude analizar el sitio.'}</p></div>
	{:else if r}
		{#if !r.connected}
			<a class="panel" href="/dashboard/conectar" style="display:block;margin-bottom:18px;text-decoration:none;color:inherit;box-shadow:var(--out),0 0 0 3px var(--accent)">
				<b>Tu blog ya funciona, pero aún no está en tu sitio.</b>
				<span class="muted"> Si abres {r.canonical_url}{r.blog_prefix} hoy verás un 404. Conéctalo en 2 minutos →</span>
			</a>
		{/if}
		<div class="kpis">
			<div class="kpi"><span>Artículos publicados</span><b>{published.length}</b></div>
			<div class="kpi"><span>Borradores por revisar</span><b>{drafts.length}</b></div>
			<div class="kpi"><span>Programados</span><b>{cal?.upcoming.length ?? 0}</b></div>
			<div class="kpi"><span>Crédito de IA disponible</span><b>US${left.toFixed(2)}</b></div>
		</div>

		{#if r.research?.sensitive && !r.auto_publish}
			<div class="panel" style="margin-bottom:18px"><b>Sector sensible ({r.research.sensitive_reason || 'salud, legal o finanzas'}).</b> <span class="muted">Google pide experiencia y revisión de un profesional en estos temas, así que los artículos quedan en borrador hasta que los apruebes.</span></div>
		{/if}

		<div class="row" style="margin-bottom:14px">
			<h2 style="font-size:1.3rem;margin:0">Artículos <span class="muted" style="font-size:.9rem;font-family:'DM Sans'">· {r.posts_per_week} por semana</span></h2>
			<label class="switch">Piloto automático <input type="checkbox" checked={r.autopilot} on:change={(e) => toggle(e.currentTarget.checked)} /></label>
		</div>
		{#if error}<p class="error" role="alert">{error}</p>{/if}

		{#if cal?.generating.length}
			<div class="panel" role="status" style="margin-bottom:14px"><span class="spin"></span>Escribiendo ahora: <b>{cal.generating[0].title}</b>{#if r.images_pending} · preparando su imagen{/if}</div>
		{/if}

		{#if recent.length}
			<div class="feed">
				{#each recent as a}
					<a class="post" href="/dashboard/articulos?abrir={a.id}">
						{#if a.has_image}<img class="thumb" src={`/api/articles/${a.id}/image`} alt="" loading="lazy" />{:else}<div class="thumb empty">{r.images_pending ? 'Creando imagen…' : 'Sin imagen'}</div>{/if}
						<div><h3>{a.title}</h3><p>{a.summary}</p><p style="margin-top:8px"><span class="chip {a.status}">{a.status === 'published' ? 'Publicado' : a.status === 'approved' ? 'Aprobado' : 'Borrador'}</span></p></div>
					</a>
				{/each}
			</div>
		{:else if !cal?.generating.length}
			<div class="panel"><p class="muted" style="margin:0">Todavía no hay artículos. {r.autopilot ? 'El primero está por salir.' : 'Enciende el piloto automático o crea uno en Artículos.'}</p></div>
		{/if}

		{#if cal?.upcoming.length}
			<h2 style="font-size:1.3rem;margin:34px 0 0">Próximos</h2>
			{#each cal.upcoming.slice(0, 4) as u}
				<div class="dayline">{day(u.at)}</div>
				<div class="post plan"><div class="thumb empty">Se creará al escribirlo</div><div><h3>{u.title}</h3><p>{intentName[u.intent] || u.intent} · palabra clave: {u.keyword}</p></div></div>
			{/each}
		{/if}
	{/if}
</div>
