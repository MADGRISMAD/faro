<script lang="ts">
	import { research, calendar, intentName } from '#lib/app';
	$: topics = $research?.research?.topics || [];
	$: planned = new Map(($calendar?.upcoming || []).map((u) => [u.title, u.at]));
	const when = (s: string) => new Date(s).toLocaleDateString('es-MX', { day: 'numeric', month: 'short' });
</script>

<div class="view">
	<h1>Palabras clave</h1>
	<p class="sub">Los temas que la IA eligió leyendo tu sitio, con la palabra clave de cada uno.</p>
	<div class="panel" style="margin-bottom:16px"><span class="muted">Todavía no mostramos volumen de búsqueda ni dificultad: eso necesita conectar una fuente de datos real (Search Console o un proveedor de palabras clave) y preferimos no inventar números.</span></div>
	<div class="panel">
		{#each topics as t}
			<div class="kw">
				<div><b>{t.title}</b><br /><span class="muted small">{t.keyword} · {t.why}</span></div>
				<div style="text-align:right;white-space:nowrap"><span class="chip">{intentName[t.intent] || t.intent}</span><br /><span class="muted small">{planned.has(t.title) ? 'Sale el ' + when(planned.get(t.title) || '') : 'Escrito o en cola'}</span></div>
			</div>
		{:else}
			<p class="muted" style="margin:0">Aún no hay temas: la IA los elige al analizar tu sitio.</p>
		{/each}
	</div>
</div>
