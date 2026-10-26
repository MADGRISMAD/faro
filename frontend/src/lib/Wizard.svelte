<script lang="ts">
	import { api } from '#lib/api';
	import { research, siteId, refreshSite, imageStyles, intentName } from '#lib/app';

	// Asistente de la primera vez: la IA propone y el cliente ajusta. Nada se escribe hasta el último paso.
	const steps = ['Tu negocio', 'Tu público', 'Tu voz', 'Imágenes', 'Temas', 'Ritmo'];
	let step = 0, busy = false, error = '', ready = false;
	let business = '', themes: string[] = [], newTheme = '', audience = '', location = '', tone = '', avoid = '';
	let images = true, style = 'clean', notes = '';
	let dropped: Record<string, boolean> = {};
	let perWeek = 2, autopilot = true, autoPublish = true;

	$: p = $research?.research;
	$: sensitive = !!p?.sensitive;
	// Los valores sugeridos se cargan una sola vez; después manda lo que el cliente va editando.
	$: if (p && !ready) {
		ready = true;
		business = p.business;
		themes = [...p.offerings];
		audience = p.audience;
		location = p.location;
		tone = p.tone;
		avoid = $research?.avoid_topics || '';
		images = $research?.images_enabled ?? true;
		style = $research?.image_style || 'clean';
		notes = $research?.image_notes || '';
		perWeek = $research?.posts_per_week || 2;
		autoPublish = !sensitive;
	}
	$: kept = (p?.topics || []).filter((t) => !dropped[t.title]).length;

	async function save(body: Record<string, unknown>) {
		await api(`/sites/${$siteId}/brand`, { method: 'PATCH', body: JSON.stringify(body) });
	}
	async function next() {
		busy = true;
		error = '';
		try {
			if (step === 0) await save({ business, themes });
			if (step === 1) await save({ audience, location });
			if (step === 2) await save({ tone, avoid_topics: avoid });
			if (step === 3) await save({ images_enabled: images, image_style: style, image_notes: notes });
			if (step === 4) await save({ remove_topics: Object.keys(dropped).filter((k) => dropped[k]) });
			step += 1;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	async function finish() {
		busy = true;
		error = '';
		try {
			await save({ auto_publish: autoPublish });
			await api(`/sites/${$siteId}/autopilot`, { method: 'PATCH', body: JSON.stringify({ enabled: autopilot, posts_per_week: perWeek }) });
			await api(`/sites/${$siteId}/brand/confirm`, { method: 'POST' });
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
			busy = false;
		}
	}
	const addTheme = () => {
		if (newTheme.trim() && themes.length < 12) themes = [...themes, newTheme.trim()];
		newTheme = '';
	};
</script>

<main class="wizard">
	<p class="eyebrow">Brand DNA · paso {step + 1} de {steps.length}</p>
	<div class="steps" aria-hidden="true">{#each steps as _, i}<i class:done={i < step} class:on={i === step}></i>{/each}</div>
	<h1>{steps[step]}</h1>
	<p class="muted" style="margin-top:0">Esto es lo que la IA entendió de <b>{$research?.domain}</b>. Está pensado para que solo corrijas lo que no te cuadre.</p>

	<div class="panel" style="margin-top:20px">
		<p class="suggested"><span class="chip">Sugerido por la IA</span> <span class="muted small">edítalo si quieres</span></p>
		{#if step === 0}
			<label>¿Qué hace tu negocio?<textarea rows="5" bind:value={business}></textarea></label>
			<div class="field" style="padding-bottom:0">
				<b>Lo que ofreces</b>
				<div class="chips">{#each themes as t, i}<button class="chip" title="Quitar" on:click={() => (themes = themes.filter((_, j) => j !== i))}>{t} ×</button>{/each}</div>
				<form style="display:flex;gap:8px" on:submit|preventDefault={addTheme}><input bind:value={newTheme} placeholder="Agregar algo que ofreces" maxlength="100" /><button class="btn ghost small">Agregar</button></form>
			</div>
		{:else if step === 1}
			<label>¿A quién le vendes?<textarea rows="4" bind:value={audience}></textarea></label>
			<label style="margin-top:14px">Ciudad o zona donde atiendes<input bind:value={location} maxlength="160" /></label>
		{:else if step === 2}
			<label>Voz y tono de tu marca<input bind:value={tone} maxlength="160" /></label>
			<label style="margin-top:14px">Temas a evitar <span class="muted small">(opcional: competidores, precios, servicios que no ofreces)</span><textarea rows="3" bind:value={avoid}></textarea></label>
		{:else if step === 3}
			<div class="row"><div><b>Crear una imagen de portada para cada artículo</b><br /><span class="muted small">Ilustración sin texto. Cuesta unos US$0.05 por artículo con su imagen.</span></div>
				<label class="switch"><input type="checkbox" bind:checked={images} /></label></div>
			{#if images}
				<div class="tiles" style="margin-top:18px">{#each imageStyles as s}<button class="tile" class:on={style === s.id} on:click={() => (style = s.id)}><b>{s.name}</b><small>{s.hint}</small></button>{/each}</div>
				<label style="margin-top:16px">Instrucciones de imagen <span class="muted small">(opcional)</span><input bind:value={notes} maxlength="300" placeholder="Por ejemplo: siempre incluir plantas, colores cálidos" /></label>
			{/if}
		{:else if step === 4}
			<p class="muted" style="margin-top:0">La IA escribirá sobre estos temas, uno por uno. Quita los que no quieras ({kept} seleccionados, mínimo 3).</p>
			{#each p?.topics || [] as t}
				<label class="kw" style="grid-template-columns:none;cursor:pointer">
					<span style="display:flex;gap:12px;align-items:flex-start"><input type="checkbox" style="width:auto;margin-top:4px" checked={!dropped[t.title]} on:change={(e) => (dropped = { ...dropped, [t.title]: !e.currentTarget.checked })} /><span><b>{t.title}</b><br /><span class="muted small">{t.why}</span></span></span>
					<span class="chip">{intentName[t.intent] || t.intent}</span>
				</label>
			{/each}
		{:else}
			<div class="field" style="padding-top:0">
				<div class="row"><div><b>Piloto automático</b><br /><span class="muted small">La IA escribe sola, a un ritmo moderado.</span></div><label class="switch"><input type="checkbox" bind:checked={autopilot} /></label></div>
			</div>
			{#if autopilot}
				<div class="field"><b>Artículos por semana</b>
					<div style="display:flex;gap:10px">{#each [1, 2, 3] as n}<button class="btn small" class:secondary={perWeek !== n} on:click={() => (perWeek = n)}>{n}</button>{/each}</div>
					<span class="muted small">Las primeras dos semanas sale 1 por semana: un ritmo constante le sienta mejor a Google que publicar mucho de golpe.</span></div>
				<div class="field">
					<div class="row"><div><b>Publicar sin que yo revise</b><br /><span class="muted small">{sensitive ? 'Tu sector es sensible (' + (p?.sensitive_reason || 'salud, legal o finanzas') + '): Google pide revisión de un profesional, así que todo queda en borrador.' : 'Si lo apagas, cada artículo queda en borrador hasta que lo apruebes.'}</span></div>
						<label class="switch"><input type="checkbox" bind:checked={autoPublish} disabled={sensitive} /></label></div>
				</div>
			{/if}
		{/if}
		{#if error}<p class="error" role="alert" style="margin-top:14px">{error}</p>{/if}
	</div>

	<div class="row" style="margin-top:22px">
		<button class="btn ghost" on:click={() => (step -= 1)} disabled={step === 0 || busy}>Atrás</button>
		{#if step < steps.length - 1}
			<button class="btn" on:click={next} disabled={busy || (step === 4 && kept < 3)}>{busy ? 'Guardando…' : 'Se ve bien, continuar'}</button>
		{:else}
			<button class="btn" on:click={finish} disabled={busy}>{busy ? 'Iniciando…' : 'Empezar a escribir'}</button>
		{/if}
	</div>
</main>

<style>
	.wizard{max-width:760px}
	.steps{display:flex;gap:8px;margin:10px 0 18px}
	.steps i{height:8px;flex:1;border-radius:99px;background:var(--well);box-shadow:var(--in-s)}
	.steps i.done{background:var(--accent-l)}.steps i.on{background:var(--accent)}
	.suggested{margin:0 0 14px;display:flex;gap:10px;align-items:center}
</style>
