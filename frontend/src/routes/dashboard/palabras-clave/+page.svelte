<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '#lib/api';
	import { siteId, intentName } from '#lib/app';

	type Metrics = { volume: number | null; competition: string | null; competition_index: number | null; cpc: number | null; difficulty: number | null };
	type Row = { title: string; keyword: string; intent: string; why: string; written: boolean; metrics?: Metrics };
	let rows: Row[] = [], configured = false, loading = true, busy = false, error = '', notice = '';

	async function load() {
		try {
			const d = await api(`/sites/${$siteId}/keywords`);
			rows = d.keywords;
			configured = d.configured;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}
	onMount(load);
	async function refresh() {
		busy = true;
		error = notice = '';
		try {
			const r = await api(`/sites/${$siteId}/keywords/refresh`, { method: 'POST' });
			notice = r.updated ? `Se actualizaron ${r.updated} palabras.` : 'Los datos ya estaban al día (se renuevan cada 30 días).';
			await load();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	const level = (d: number) => (d < 30 ? 'easy' : d < 60 ? 'medium' : 'hard');
	const levelName: Record<string, string> = { easy: 'fácil', medium: 'media', hard: 'difícil' };
	$: withData = rows.filter((r) => r.metrics?.volume != null).length;
</script>

<div class="view">
	<div class="row"><div><h1>Palabras clave</h1><p class="sub">Los temas que la IA eligió, ordenados por oportunidad real: mucho volumen y poca dificultad.</p></div>
		{#if configured}<button class="btn ghost" on:click={refresh} disabled={busy}>{busy ? 'Actualizando…' : 'Actualizar datos'}</button>{/if}</div>
	{#if notice}<p class="success" role="status">{notice}</p>{/if}
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if !loading && !configured}
		<div class="panel" style="margin-bottom:16px"><b>Sin datos de búsqueda todavía.</b> <span class="muted">Para ver volumen mensual, dificultad y costo por clic reales hay que conectar un proveedor de palabras clave (DataForSEO) en el servidor; mientras tanto no se muestra ningún número para no inventarlos. Pasos en <code>docs/DATOS.md</code>.</span></div>
	{:else if !loading && withData === 0 && rows.length}
		<div class="panel" style="margin-bottom:16px"><span class="muted">Aún no hay datos para estas palabras. Pulsa «Actualizar datos».</span></div>
	{/if}
	<div class="panel">
		{#if loading}<span class="spin"></span>Cargando…
		{:else}
			<div style="overflow-x:auto"><table class="data"><thead><tr><th>Tema</th><th class="r">Búsquedas / mes</th><th class="r">Dificultad</th><th class="r">Costo por clic</th><th>Estado</th></tr></thead><tbody>
				{#each rows as r}
					<tr>
						<td><b>{r.title}</b><br /><span class="muted small">{r.keyword} · {intentName[r.intent] || r.intent}</span></td>
						<td class="r">{r.metrics?.volume != null ? r.metrics.volume.toLocaleString('es-MX') : '—'}</td>
						<td class="r">{#if r.metrics?.difficulty != null}<span class="kd {level(r.metrics.difficulty)}">{r.metrics.difficulty}</span> <span class="muted small">{levelName[level(r.metrics.difficulty)]}</span>{:else}—{/if}</td>
						<td class="r">{r.metrics?.cpc != null ? 'US$' + r.metrics.cpc.toFixed(2) : '—'}</td>
						<td>{r.written ? 'Escrito o en cola' : 'Pendiente'}</td>
					</tr>
				{:else}<tr><td colspan="5" class="muted">Aún no hay temas: la IA los elige al analizar tu sitio.</td></tr>{/each}
			</tbody></table></div>
			{#if withData}<p class="muted small" style="margin-bottom:0">Fuente: Google Ads vía DataForSEO, México y español. La dificultad va de 0 a 100 (probabilidad de entrar al top 10).</p>{/if}
		{/if}
	</div>
</div>
