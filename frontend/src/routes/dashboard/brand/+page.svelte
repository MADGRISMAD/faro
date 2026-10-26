<script lang="ts">
	import { api } from '#lib/api';
	import { research, siteId, sites, refreshSite, imageStyles } from '#lib/app';

	let editing = '', draft = '', error = '', saved = '', newTheme = '', notes = '', notesSite = '';
	$: p = $research?.research;
	$: domain = $sites.find((s) => s.id === $siteId)?.domain || '';
	$: reading = $research?.status === 'queued' || $research?.status === 'running';
	$: style = imageStyles.find((s) => s.id === $research?.image_style) || imageStyles[0];
	// las instrucciones de imagen se editan en línea; se recargan al cambiar de sitio
	$: if ($siteId !== notesSite && $research) {
		notesSite = $siteId;
		notes = $research.image_notes || '';
	}

	async function patch(body: Record<string, unknown>) {
		error = '';
		try {
			await api(`/sites/${$siteId}/brand`, { method: 'PATCH', body: JSON.stringify(body) });
			saved = 'Guardado';
			setTimeout(() => (saved = ''), 1500);
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		}
	}
	function edit(key: string, value: string) {
		editing = key;
		draft = value;
	}
	async function commit() {
		await patch({ [editing]: draft });
		editing = '';
	}
	const addTheme = async () => {
		if (!newTheme.trim() || !p) return;
		await patch({ themes: [...p.offerings, newTheme.trim()] });
		newTheme = '';
	};
	async function reread() {
		if (!confirm(`La IA volverá a leer ${domain} y reemplazará lo que ves aquí (tipo de negocio, tono, público y temas). ¿Continuar?`)) return;
		error = '';
		try {
			await api(`/sites/${$siteId}/research`, { method: 'POST' });
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		}
	}

	// «Profesional, innovador y accesible» → chips; un «en México» al final se muestra aparte
	const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
	function split(text: string) {
		const parts = text.replace(/\.$/, '').split(/,\s*|\s+y\s+|;\s*/).map((s) => s.trim()).filter(Boolean);
		let where = '';
		const m = parts.length ? parts[parts.length - 1].match(/^(.*\S)\s+en\s+(.+)$/) : null;
		if (m) {
			parts[parts.length - 1] = m[1];
			where = 'En ' + m[2];
		}
		return { items: parts.map(cap), where };
	}
	$: tone = split(p?.tone || '');
	$: audience = split(p?.audience || '');
</script>

<!-- Íconos de línea de las tarjetas -->
{#snippet icon(d: string)}
	<span class="ico"><svg viewBox="0 0 24 24" aria-hidden="true"><path {d} /></svg></span>
{/snippet}

{#snippet editBtn(key: string, value: string)}
	{#if editing === key}
		<span class="acts"><button class="btn ghost small" on:click={() => (editing = '')}>Cancelar</button><button class="btn small" on:click={commit}>Guardar</button></span>
	{:else}
		<button class="btn ghost small" on:click={() => edit(key, value)}>{value ? 'Editar' : 'Agregar'}</button>
	{/if}
{/snippet}

<div class="view wide brand">
	<div class="top">
		<div><h1>Brand DNA</h1><p class="sub">Así entiende la IA tu marca. Todo lo que escribe sale de aquí.</p></div>
		<div class="top-acts">
			{#if saved}<span class="chip published">{saved}</span>{/if}
			{#if p}<span class="learned" class:busy={reading}><i></i>{reading ? `Leyendo ${domain}…` : `Aprendido de ${domain}`}</span>{/if}
			<button class="btn ghost small" disabled={reading} on:click={reread}><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 12a8 8 0 1 1-2.3-5.6M20 4v5h-5" /></svg>Volver a leer mi sitio</button>
		</div>
	</div>
	{#if error}<p class="error" role="alert">{error}</p>{/if}

	{#if !p && $research?.status === 'none'}
		<div class="panel">Este sitio se dio de alta a mano, así que la IA no lo ha leído todavía. Usa «Volver a leer mi sitio» y la IA llenará aquí todo por ti.</div>
	{:else if !p}
		<div class="panel"><span class="spin"></span>Todavía estoy analizando tu sitio…</div>
	{:else}
		<!-- Tipo de negocio, idioma y zona -->
		<section class="panel hero-card">
			<div class="biz">
				<div class="head"><p class="eyebrow">Tipo de negocio</p>{@render editBtn('business', p.business)}</div>
				{#if editing === 'business'}<textarea bind:value={draft} rows="4"></textarea>{:else}<p class="biz-text">{p.business}</p>{/if}
			</div>
			<div class="facts">
				<div class="fact">
					{@render icon('M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Zm-9 9h18M12 3c2.5 2.6 3.7 5.6 3.7 9s-1.2 6.4-3.7 9c-2.5-2.6-3.7-5.6-3.7-9S9.5 5.6 12 3Z')}
					<div><small>Idioma del contenido</small><b>Español (México)</b></div>
				</div>
				<div class="fact">
					{@render icon('M12 21s-7-6.2-7-11.5a7 7 0 0 1 14 0C19 14.8 12 21 12 21Zm0-9a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z')}
					<div class="grow"><small>Zona de servicio · opcional</small>{#if editing === 'location'}<input bind:value={draft} maxlength="120" placeholder="Ciudad o zona que atiendes" />{:else}<b>{p.location || 'Sin zona'}</b>{/if}</div>
					{@render editBtn('location', p.location || '')}
				</div>
			</div>
		</section>

		<!-- Voz y público -->
		<div class="two">
			<section class="panel">
				<div class="card-head">{@render icon('M4 5h16v11H9l-5 4V5Z')}<div class="grow"><h2>Voz y tono</h2><small>Cómo suenan tus artículos</small></div>{@render editBtn('tone', p.tone)}</div>
				{#if editing === 'tone'}<textarea bind:value={draft} rows="3"></textarea>
				{:else}<div class="pills">{#each tone.items as t}<span class="pill-tone"><i></i>{t}</span>{/each}</div>{/if}
			</section>
			<section class="panel">
				<div class="card-head">{@render icon('M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-6 9a6 6 0 0 1 12 0M16 3.5a4 4 0 0 1 0 7.5M21 20a6 6 0 0 0-3.5-5.5')}<div class="grow"><h2>Público objetivo</h2><small>Para quién escribimos</small></div>{@render editBtn('audience', p.audience)}</div>
				{#if editing === 'audience'}<textarea bind:value={draft} rows="3"></textarea>
				{:else}<div class="pills">{#each audience.items as a}<span class="pill-aud">{a}</span>{/each}</div>{#if audience.where}<p class="where">{audience.where}</p>{/if}{/if}
			</section>
		</div>

		<!-- Temas -->
		<div class="two themes">
			<section class="panel">
				<div class="card-head">{@render icon('M3 12V4h8l10 10-8 8L3 12Zm5-4h.01')}<div class="grow"><h2>Temas clave <span class="count">· {p.offerings.length}</span></h2><small>Tus artículos giran alrededor de estos temas</small></div></div>
				<div class="pills">
					{#each p.offerings as t, i}<span class="theme">{t}<button aria-label="Quitar {t}" on:click={() => patch({ themes: p.offerings.filter((_, j) => j !== i) })}>×</button></span>{/each}
				</div>
				<form class="add" on:submit|preventDefault={addTheme}><input bind:value={newTheme} placeholder="Agregar tema clave…" maxlength="100" aria-label="Nuevo tema clave" /><button class="btn small" disabled={!newTheme.trim()}>Agregar</button></form>
			</section>
			<section class="panel">
				<div class="card-head">{@render icon('M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18ZM5.6 5.6l12.8 12.8')}<div class="grow"><h2>Temas a evitar</h2><small>Opcional</small></div>{#if $research?.avoid_topics || editing === 'avoid_topics'}{@render editBtn('avoid_topics', $research?.avoid_topics || '')}{/if}</div>
				<div class="avoid">
					{#if editing === 'avoid_topics'}<textarea bind:value={draft} rows="4" placeholder="Por ejemplo: nombres de competidores, precios, servicios que no ofreces"></textarea>
					{:else if $research?.avoid_topics}<p class="avoid-text">{$research.avoid_topics}</p><small>La IA nunca mencionará esto.</small>
					{:else}
						<p>La IA nunca mencionará lo que pongas aquí.</p>
						<small>Por ejemplo: nombres de competidores, precios, servicios que no ofreces.</small>
						<button class="btn ghost small" on:click={() => edit('avoid_topics', '')}>+ Agregar tema a evitar</button>
					{/if}
				</div>
			</section>
		</div>

		<!-- Imágenes -->
		<section class="panel images">
			<div class="card-head">{@render icon('M4 5h16v14H4V5Zm0 11 4.5-4.5 3.5 3.5 2.5-2.5L20 17M15.5 9.5h.01')}<div class="grow"><h2>Imágenes de tus artículos</h2><small>Una portada por artículo, sin texto. Cuesta unos US$0.045 por imagen.</small></div>
				<label class="switch">{$research?.images_enabled ? 'Activado' : 'Apagado'}<input type="checkbox" checked={$research?.images_enabled} on:change={(e) => patch({ images_enabled: e.currentTarget.checked })} /></label>
			</div>
			{#if $research?.images_enabled}
				<div class="style-head"><p class="eyebrow">Estilo de imagen</p><span>Elegido: <b>{style.name}</b></span></div>
				<div class="styles">
					{#each imageStyles as s}
						<button class="style" class:on={$research?.image_style === s.id} aria-pressed={$research?.image_style === s.id} on:click={() => patch({ image_style: s.id })}>
							<span class="shot"><img src="/estilos/{s.id}.jpg" width="480" height="270" loading="lazy" alt="Muestra del estilo {s.name}" />{#if $research?.image_style === s.id}<i class="check"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5" /></svg></i>{/if}</span>
							<b>{s.name}</b><small>{s.hint}</small>
						</button>
					{/each}
				</div>
				<form class="notes" on:submit|preventDefault={() => patch({ image_notes: notes })}>
					<label for="notes"><b>Instrucciones de imagen</b> <span>· opcional</span></label>
					<div class="row-in"><input id="notes" bind:value={notes} maxlength="300" placeholder="Por ejemplo: siempre incluir plantas, colores cálidos" /><button class="btn ghost small" disabled={notes === ($research?.image_notes || '')}>Guardar</button></div>
				</form>
			{/if}
		</section>
	{/if}
</div>

<style>
	.brand { display: grid; gap: 26px; }
	.top { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
	.top h1 { font-size: 2.4rem; }
	.top .sub { margin: 6px 0 0; }
	.top-acts { display: flex; gap: 12px; align-items: center; flex-wrap: wrap; }
	.top-acts .btn { gap: 8px; }
	.top-acts svg, .ico svg, .check svg { width: 18px; height: 18px; fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; stroke-linejoin: round; }
	.learned { display: inline-flex; align-items: center; gap: 8px; padding: 8px 16px; border-radius: 99px; box-shadow: var(--out-s); font-size: .84rem; font-weight: 800; color: #4a3a4e; }
	.learned i { width: 9px; height: 9px; border-radius: 50%; background: #16a34a; }
	.learned.busy i { background: var(--accent); animation: blink 1s infinite; }
	@keyframes blink { 50% { opacity: .3 } }

	.eyebrow { margin: 0; color: #5d4d61; }
	.head, .card-head { display: flex; align-items: center; gap: 14px; }
	.head { justify-content: space-between; }
	.grow { flex: 1; min-width: 0; }
	.card-head { margin-bottom: 20px; }
	.card-head h2 { margin: 0; font-size: 1.2rem; }
	.card-head small { color: var(--mut); font-size: .84rem; }
	.count { color: var(--accent); font-size: .8em; }
	.ico { display: grid; place-items: center; width: 44px; height: 44px; border-radius: 15px; box-shadow: var(--out-s); color: var(--accent); flex: none; }
	.acts { display: flex; gap: 8px; }
	textarea, .fact input { margin-top: 4px; }

	.hero-card { display: grid; grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr); gap: 34px; padding: 32px; align-items: start; }
	.biz-text { margin: 18px 0 0; font-size: 1.45rem; line-height: 1.4; font-weight: 900; letter-spacing: -.015em; }
	.facts { display: grid; gap: 16px; }
	.fact { display: flex; align-items: center; gap: 14px; padding: 14px 18px; border-radius: 26px; box-shadow: var(--in-s); }
	.fact .ico { width: 38px; height: 38px; border-radius: 50%; }
	.fact small { display: block; color: var(--mut); font-size: .76rem; font-weight: 700; }
	.fact b { font-size: 1rem; }

	.two { display: grid; grid-template-columns: 1fr 1fr; gap: 26px; }
	.two.themes { grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr); }
	.two .panel, .images { padding: 26px; }
	.pills { display: flex; flex-wrap: wrap; gap: 12px; }
	.pill-tone { display: inline-flex; align-items: center; gap: 9px; padding: 11px 20px; border-radius: 99px; box-shadow: var(--out-s); font-weight: 800; }
	.pill-tone i { width: 7px; height: 7px; border-radius: 50%; background: var(--accent); }
	.pill-aud { padding: 8px 16px; border-radius: 99px; box-shadow: var(--in-s); font-weight: 700; font-size: .92rem; }
	.where { margin: 16px 0 0; color: var(--mut); font-size: .86rem; }
	.theme { display: inline-flex; align-items: center; gap: 10px; padding: 7px 8px 7px 15px; border-radius: 99px; background: var(--accent-xl); color: var(--accent-d); font-weight: 800; font-size: .86rem; }
	.theme button { border: 0; width: 22px; height: 22px; border-radius: 50%; background: var(--surface); color: var(--accent-d); font-weight: 800; line-height: 1; padding: 0; box-shadow: var(--out-s); }
	.theme button:hover { background: var(--accent); color: #fff; }
	.add { display: flex; gap: 12px; margin-top: 22px; }
	.add .btn { box-shadow: var(--puff); }
	.avoid { display: grid; align-content: center; gap: 10px; min-height: 260px; padding: 22px; border-radius: 26px; box-shadow: var(--in-s); }
	.avoid p { margin: 0; color: #4a3a4e; }
	.avoid small { color: var(--mut); line-height: 1.5; }
	.avoid .btn { justify-self: start; margin-top: 6px; }
	.avoid-text { white-space: pre-wrap; font-weight: 700; }

	.images .card-head { margin-bottom: 26px; }
	.switch { gap: 14px; font-size: .92rem; }
	.style-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; font-size: .86rem; color: #4a3a4e; }
	.style-head b { color: var(--accent); }
	.styles { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 22px; }
	.style { border: 0; text-align: left; padding: 10px 10px 16px; border-radius: 26px; background: var(--surface); box-shadow: var(--out-s); color: var(--ink); display: grid; gap: 4px; transition: transform .15s; }
	.style:hover { transform: translateY(-2px); }
	.style.on { background: var(--accent-xl); box-shadow: var(--out-s), 0 0 0 3px var(--accent); }
	.shot { position: relative; display: block; margin-bottom: 10px; }
	.shot img { width: 100%; height: auto; aspect-ratio: 16 / 9; object-fit: cover; border-radius: 16px; display: block; }
	.check { position: absolute; top: 10px; right: 10px; display: grid; place-items: center; width: 30px; height: 30px; border-radius: 50%; background: var(--accent); color: #fff; box-shadow: 0 4px 10px rgba(194, 24, 91, .35); }
	.check svg { width: 16px; height: 16px; stroke-width: 3; }
	.style b { padding: 0 6px; font-size: 1rem; }
	.style small { padding: 0 6px; color: var(--mut); font-size: .82rem; }
	.notes { margin-top: 26px; padding: 18px 22px; border-radius: 26px; box-shadow: var(--in-s); display: grid; gap: 8px; }
	.notes label { display: block; font-size: .95rem; }
	.notes label span { color: var(--accent); font-size: .8rem; font-weight: 800; }
	.row-in { display: flex; gap: 12px; align-items: center; }
	.row-in input { box-shadow: none; padding-left: 0; background: transparent; }

	@media (max-width: 1100px) {
		.styles { grid-template-columns: repeat(2, minmax(0, 1fr)); }
		.hero-card, .two, .two.themes { grid-template-columns: minmax(0, 1fr); }
	}
	@media (max-width: 560px) {
		.hero-card, .two .panel, .images { padding: 20px; }
		.biz-text { font-size: 1.15rem; }
		.styles { gap: 12px; }
		.add { flex-direction: column; }
		.images .card-head { flex-wrap: wrap; }
		.images .card-head .grow { flex-basis: calc(100% - 60px); }
		.images .switch { margin-left: 58px; }
		.style-head { flex-direction: column; align-items: flex-start; gap: 6px; }
	}
	@media (prefers-reduced-motion: reduce) { .learned.busy i { animation: none; } .style { transition: none; } }
</style>
