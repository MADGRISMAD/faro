<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '#lib/api';
	import { siteId } from '#lib/app';

	type Totals = { clicks: number; impressions: number; ctr: number; position: number };
	type Data = {
		configured: boolean; connected: boolean; property?: string; error?: string; range?: string; start?: string; end?: string;
		history?: { days_with_data: number; first?: string; last?: string } | null;
		totals?: Totals; previous?: Totals; series?: { date: string; clicks: number; impressions: number }[];
		queries?: { query: string; clicks: number; impressions: number; ctr: number; position: number }[];
		pages?: { page: string; clicks: number; impressions: number; ctr: number; position: number }[];
	};
	let range = '28d', data: Data | null = null, loading = true, error = '', notice = '', busy = false;
	const notes: Record<string, string> = { ok: 'Google conectado.', denied: 'No diste permiso en Google, así que no se conectó nada.', noproperty: 'Conectado, pero esa cuenta de Google no tiene este sitio verificado en Search Console.', error: 'No pude completar la conexión con Google. Inténtalo de nuevo.' };

	async function load() {
		loading = true;
		error = '';
		try {
			data = await api(`/sites/${$siteId}/analytics?range=${range}`);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}
	onMount(() => {
		const g = new URLSearchParams(location.search).get('google');
		if (g) notice = notes[g] || '';
		load();
	});
	async function connect() {
		busy = true;
		error = '';
		try {
			const r = await api(`/sites/${$siteId}/google/start`, { method: 'POST' });
			location.href = r.url;
		} catch (e) {
			error = (e as Error).message;
			busy = false;
		}
	}
	async function disconnect() {
		busy = true;
		try {
			await api(`/sites/${$siteId}/google`, { method: 'DELETE' });
			await load();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	const n = (v: number) => Math.round(v).toLocaleString('es-MX');
	const pct = (v: number) => (v * 100).toFixed(1) + ' %';
	// Cambio contra el periodo anterior; en posición, bajar es mejorar.
	function delta(cur: number, prev: number, lowerIsBetter = false) {
		if (!prev) return cur ? { text: 'nuevo', good: true } : { text: '—', good: true };
		const d = ((cur - prev) / prev) * 100;
		return { text: (d >= 0 ? '+' : '') + d.toFixed(0) + ' %', good: lowerIsBetter ? d <= 0 : d >= 0 };
	}
	function spark(values: number[]) {
		if (values.length < 2) return '';
		const max = Math.max(...values, 1);
		return values.map((v, i) => `${(i / (values.length - 1)) * 100},${30 - (v / max) * 28}`).join(' ');
	}
	let t: Totals | undefined, p: Totals | undefined;
	$: t = data?.totals;
	$: p = data?.previous;
	$: cards = t && p ? [
		['Impresiones', n(t.impressions), delta(t.impressions, p.impressions), 'Veces que Google te mostró', (data?.series || []).map((x) => x.impressions)],
		['Clics', n(t.clicks), delta(t.clicks, p.clicks), 'Visitas que llegaron desde Google', (data?.series || []).map((x) => x.clicks)],
		['CTR', pct(t.ctr), delta(t.ctr, p.ctr), 'Qué tan seguido hacen clic', []],
		['Posición media', t.position ? '#' + t.position.toFixed(1) : '—', delta(t.position, p.position, true), 'Tu lugar promedio en Google', []]
	] as [string, string, { text: string; good: boolean }, string, number[]][] : [];
</script>

<div class="view wide">
	<div class="row"><div><h1>Analytics</h1><p class="sub">Datos reales de Google Search Console{#if data?.start} · del {data.start} al {data.end}{/if}</p></div>
		{#if data?.connected && data.property}<div style="display:flex;gap:8px">{#each ['7d', '28d', '90d'] as r}<button class="btn small" class:secondary={range !== r} on:click={() => { range = r; load(); }}>{r}</button>{/each}</div>{/if}
	</div>
	{#if notice}<p class="success" role="status">{notice}</p>{/if}
	{#if error}<p class="error" role="alert">{error}</p>{/if}

	{#if loading}
		<div class="panel"><span class="spin"></span>Pidiendo tus datos a Google…</div>
	{:else if data && !data.connected}
		<div class="panel" style="max-width:640px">
			<h2 style="font-size:1.4rem;margin:0 0 6px">Conecta Google Search Console</h2>
			<p class="muted" style="margin-top:0">Así ves cuántas veces Google te muestra, cuántos clics recibes y en qué lugar sales, con los datos reales de Google. Solo lectura: Faro no puede cambiar nada de tu cuenta. Lee la <a href="https://www.hyperionmkt.com/faro/privacidad" target="_blank" rel="noopener" style="text-decoration:underline">política de privacidad</a>.</p>
			<ul class="muted" style="padding-left:18px;line-height:1.7"><li>Tu sitio debe estar verificado en <b>search.google.com/search-console</b> con esa cuenta.</li><li>Google tarda unos 2-3 días en reportar, y un sitio nuevo necesita semanas para aparecer.</li></ul>
			{#if data.configured}
				<button class="btn" on:click={connect} disabled={busy}>{busy ? 'Abriendo Google…' : 'Conectar con Google'}</button>
			{:else}
				<p class="error" role="status">Esta función todavía no está configurada en este servidor: faltan las credenciales de Google (ver <code>docs/DATOS.md</code>).</p>
			{/if}
		</div>
	{:else if data && !data.property}
		<div class="panel" style="max-width:640px"><p class="error" role="alert">{data.error}</p>
			<div style="display:flex;gap:10px"><button class="btn" on:click={connect} disabled={busy}>Conectar otra cuenta</button><button class="btn ghost" on:click={disconnect} disabled={busy}>Desconectar</button></div></div>
	{:else if data}
		<div class="kpis">
			{#each cards as [label, value, d, hint, series]}
				<div class="kpi"><span>{label}</span><b>{value}</b>
					<span class="small" style="font-weight:800;color:{d.good ? 'var(--ink)' : 'var(--accent-d)'}">{d.text} <span class="muted" style="font-weight:500">vs periodo anterior</span></span>
					{#if series.length > 1}<svg viewBox="0 0 100 32" preserveAspectRatio="none" style="width:100%;height:34px;margin-top:8px" aria-hidden="true"><polyline points={spark(series)} fill="none" stroke="var(--accent)" stroke-width="2" stroke-linejoin="round" vector-effect="non-scaling-stroke" /></svg>{:else}<span class="muted small">{hint}</span>{/if}
				</div>
			{/each}
		</div>
		{#if !(data.totals?.impressions)}
			<div class="panel" style="margin-bottom:18px">
				<b>La conexión funciona, pero Google no reporta impresiones en este periodo.</b>
				{#if data.history && data.history.days_with_data > 0}
					<p class="muted" style="margin:6px 0 0">Google sí tiene datos de este sitio: {data.history.days_with_data} días entre el {data.history.first} y el {data.history.last}. Prueba con otro periodo (90d).</p>
				{:else}
					<p class="muted" style="margin:6px 0 0">En los últimos 16 meses Google no tiene ni una impresión registrada para <b>{data.property}</b>. Es lo normal si la propiedad se verificó hace poco, si el sitio es nuevo o si todavía casi no sale en búsquedas: Search Console empieza a llenarse uno o dos días después de verificar y un sitio nuevo tarda semanas en aparecer. Faro consulta a Google en vivo cada hora, así que los números aparecerán solos.</p>
				{/if}
			</div>
		{/if}
		<div class="panel" style="margin-bottom:18px">
			<h2 style="font-size:1.2rem;margin:0 0 10px">Palabras con las que te encuentran</h2>
			<div style="overflow-x:auto"><table class="data"><thead><tr><th>Búsqueda</th><th class="r">Clics</th><th class="r">Impresiones</th><th class="r">CTR</th><th class="r">Posición</th></tr></thead><tbody>
				{#each data.queries || [] as q}<tr><td>{q.query}</td><td class="r">{n(q.clicks)}</td><td class="r">{n(q.impressions)}</td><td class="r">{pct(q.ctr)}</td><td class="r">#{q.position.toFixed(1)}</td></tr>{:else}<tr><td colspan="5" class="muted">Sin datos todavía.</td></tr>{/each}
			</tbody></table></div>
		</div>
		<div class="panel" style="margin-bottom:18px">
			<h2 style="font-size:1.2rem;margin:0 0 10px">Páginas más vistas en Google</h2>
			<div style="overflow-x:auto"><table class="data"><thead><tr><th>Página</th><th class="r">Clics</th><th class="r">Impresiones</th><th class="r">CTR</th><th class="r">Posición</th></tr></thead><tbody>
				{#each data.pages || [] as q}<tr><td>{q.page.replace(/^https?:\/\/[^/]+/, '') || '/'}</td><td class="r">{n(q.clicks)}</td><td class="r">{n(q.impressions)}</td><td class="r">{pct(q.ctr)}</td><td class="r">#{q.position.toFixed(1)}</td></tr>{:else}<tr><td colspan="5" class="muted">Sin datos todavía.</td></tr>{/each}
			</tbody></table></div>
		</div>
		<p class="muted small">Propiedad: {data.property} · <button class="btn ghost small" on:click={disconnect} disabled={busy}>Desconectar Google</button></p>
	{/if}
</div>
