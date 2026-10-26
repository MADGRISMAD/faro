<script lang="ts">
	import { api } from '#lib/api';
	import { research, siteId, refreshSite, imageStyles } from '#lib/app';

	let editing = '', draft = '', error = '', saved = '', newTheme = '';
	$: p = $research?.research;

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
	const fields: [string, string, () => string][] = [
		['business', 'Tipo de negocio', () => p?.business || ''],
		['tone', 'Voz y tono', () => p?.tone || ''],
		['audience', 'Público objetivo', () => p?.audience || ''],
		['location', 'Zona de servicio', () => p?.location || ''],
		['avoid_topics', 'Temas a evitar', () => $research?.avoid_topics || '']
	];
	const hints: Record<string, string> = { location: 'Si tu negocio atiende una ciudad o zona específica, agrégala aquí', avoid_topics: 'Por ejemplo: nombres de competidores, precios, servicios que no ofreces' };
	const addTheme = async () => {
		if (!newTheme.trim() || !p) return;
		await patch({ themes: [...p.offerings, newTheme.trim()] });
		newTheme = '';
	};
</script>

<div class="view">
	<div class="row"><div><h1>Brand DNA</h1><p class="sub">La IA aprende tu marca leyendo tu sitio. Puedes ajustarla cuando quieras.</p></div>{#if saved}<span class="chip published">{saved}</span>{/if}</div>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if !p && $research?.status === 'none'}
		<div class="panel">Este sitio se dio de alta a mano, así que la IA no lo ha leído todavía. Añade el sitio otra vez pegando su dirección y la IA llenará aquí todo por ti.</div>
	{:else if !p}
		<div class="panel"><span class="spin"></span>Todavía estoy analizando tu sitio…</div>
	{:else}
		<div class="panel">
			<div class="field"><div class="head"><b>Idioma del contenido</b></div><p>Español (México)</p></div>
			{#each fields as [key, label, get]}
				<div class="field">
					<div class="head"><b>{label}{#if hints[key]} <span class="chip" style="font-weight:500">Opcional</span>{/if}</b>
						{#if editing === key}<button class="btn small" on:click={commit}>Guardar</button>{:else}<button class="btn ghost small" on:click={() => edit(key, get())}>{get() ? 'Editar' : 'Agregar'}</button>{/if}
					</div>
					{#if editing === key}<textarea bind:value={draft} rows="4"></textarea>{:else if get()}<p>{get()}</p>{:else}<p class="muted"><i>{hints[key]}</i></p>{/if}
				</div>
			{/each}
			<div class="field">
				<div class="head"><b>Temas clave</b></div>
				<div class="chips">
					{#each p.offerings as t, i}<button class="chip" title="Quitar" on:click={() => patch({ themes: p.offerings.filter((_, j) => j !== i) })}>{t} ×</button>{/each}
				</div>
				<form style="display:flex;gap:8px" on:submit|preventDefault={addTheme}><input bind:value={newTheme} placeholder="Agregar tema clave" maxlength="100" /><button class="btn ghost small">Agregar</button></form>
			</div>
			<div class="field">
				<div class="head"><div><b>Crear imágenes para mis artículos</b><br /><span class="muted small">Una portada por artículo, sin texto. Cuesta unos US$0.045 por imagen.</span></div>
					<label class="switch"><input type="checkbox" checked={$research?.images_enabled} on:change={(e) => patch({ images_enabled: e.currentTarget.checked })} /></label></div>
			</div>
			{#if $research?.images_enabled}
				<div class="field">
					<div class="head"><b>Instrucciones de imagen <span class="chip" style="font-weight:500">Opcional</span></b>
						{#if editing === 'image_notes'}<button class="btn small" on:click={commit}>Guardar</button>{:else}<button class="btn ghost small" on:click={() => edit('image_notes', $research?.image_notes || '')}>{$research?.image_notes ? 'Editar' : 'Agregar'}</button>{/if}</div>
					{#if editing === 'image_notes'}<textarea bind:value={draft} rows="3"></textarea>{:else if $research?.image_notes}<p>{$research.image_notes}</p>{:else}<p class="muted"><i>Por ejemplo: siempre incluir plantas, colores cálidos</i></p>{/if}
				</div>
				<div class="field">
					<div class="head"><b>Estilo de imagen</b></div>
					<div class="tiles">
						{#each imageStyles as s}
							<button class="tile" class:on={$research?.image_style === s.id} on:click={() => patch({ image_style: s.id })}><b>{s.name}</b><small>{s.hint}</small></button>
						{/each}
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>
