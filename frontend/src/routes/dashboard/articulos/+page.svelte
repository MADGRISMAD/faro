<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '#lib/api';
	import { articles, sites, siteId, research, refreshSite, statusName, type Article } from '#lib/app';

	let selected: Article | null = null;
	let error = '', notice = '', busy = false, topic = '', showNew = false;
	$: published = $articles.filter((a) => a.status === 'published').length;
	$: site = $sites.find((s) => s.id === $siteId);
	// la portada llega después del texto: si el artículo abierto ya la tiene, se muestra sin recargar
	$: if (selected && !selected.has_image && $articles.some((x) => x.id === selected?.id && x.has_image)) selected.has_image = true;

	onMount(async () => {
		const id = new URLSearchParams(location.search).get('abrir');
		if (id) await open(id);
	});
	async function open(id: string) {
		error = '';
		selected = await api(`/articles/${id}`);
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}
	async function save() {
		if (!selected) return;
		busy = true;
		error = '';
		try {
			await api(`/articles/${selected.id}`, { method: 'PATCH', body: JSON.stringify({ title: selected.title, slug: selected.slug, meta_description: selected.meta_description, summary: selected.summary, content: selected.content }) });
			notice = 'Cambios guardados.';
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	async function move(action: 'approve' | 'publish') {
		if (!selected) return;
		busy = true;
		error = '';
		try {
			await api(`/articles/${selected.id}/${action}`, { method: 'POST' });
			selected = await api(`/articles/${selected.id}`);
			notice = action === 'approve' ? 'Artículo aprobado.' : 'Artículo publicado.';
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	async function create() {
		busy = true;
		error = '';
		try {
			const p = $research?.research;
			await api(`/sites/${$siteId}/generations`, { method: 'POST', headers: { 'idempotency-key': crypto.randomUUID() }, body: JSON.stringify({ topic, business: p?.business || site?.brand_description || site?.name, audience: p?.audience || 'Clientes potenciales', keyword: null }) });
			topic = '';
			showNew = false;
			notice = 'La IA está escribiendo tu artículo. Aparece aquí en un momento.';
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	const url = (a: Article) => ($research?.blog_base ? `${$research.blog_base}/${a.slug}` : '#');
	const lock = () => selected?.status === 'published';
</script>

<div class="view">
	<div class="row"><div><h1>Artículos</h1><p class="sub">Administra tu contenido SEO</p></div><button class="btn ghost" on:click={() => (showNew = !showNew)}>+ Nuevo artículo</button></div>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if notice}<p class="success" role="status">{notice}</p>{/if}

	{#if showNew}
		<form class="panel" style="margin-bottom:20px;display:flex;gap:10px;flex-wrap:wrap" on:submit|preventDefault={create}>
			<input bind:value={topic} required maxlength="300" placeholder="¿De qué quieres escribir?" style="flex:1;min-width:240px" />
			<button class="btn" disabled={busy}>Escribir con IA</button>
		</form>
	{/if}

	<div class="kpis">
		<div class="kpi"><span>Total</span><b>{$articles.length}</b></div>
		<div class="kpi"><span>Borradores</span><b>{$articles.length - published}</b></div>
		<div class="kpi"><span>Publicados</span><b>{published}</b></div>
	</div>

	{#if selected}
		<section class="panel" style="margin-bottom:24px">
			<div class="row"><h2 style="margin:0;font-size:1.4rem">{selected.title}</h2><button class="btn secondary small" on:click={() => (selected = null)}>Cerrar</button></div>
			{#if selected.has_image}<img class="thumb" style="margin:16px 0" src={`/api/articles/${selected.id}/image`} alt="Portada del artículo" />{/if}
			<div class="stack" style="margin-top:12px">
				<label>Título<input bind:value={selected.title} disabled={lock()} /></label>
				<label>Slug<input bind:value={selected.slug} disabled={lock()} /></label>
				<label>Meta description<textarea bind:value={selected.meta_description} disabled={lock()}></textarea></label>
				<label>Resumen<textarea bind:value={selected.summary} disabled={lock()}></textarea></label>
				<label>Introducción<textarea bind:value={selected.content.intro} disabled={lock()}></textarea></label>
				{#each selected.content.sections as section}
					<div class="card">
						<label>Encabezado<input bind:value={section.heading} disabled={lock()} /></label>
						{#each section.paragraphs as _, i}<label>Párrafo<textarea bind:value={section.paragraphs[i]} disabled={lock()}></textarea></label>{/each}
						{#each section.bullets as _, i}<label>Viñeta<input bind:value={section.bullets[i]} disabled={lock()} /></label>{/each}
					</div>
				{/each}
				{#each selected.content.faq as faq}
					<div class="card"><label>Pregunta<input bind:value={faq.question} disabled={lock()} /></label><label>Respuesta<textarea bind:value={faq.answer} disabled={lock()}></textarea></label></div>
				{/each}
				<div style="display:flex;gap:10px;flex-wrap:wrap">
					{#if selected.status !== 'published'}<button class="btn secondary" on:click={save} disabled={busy}>Guardar cambios</button>{/if}
					{#if selected.status === 'draft'}<button class="btn" on:click={() => move('approve')} disabled={busy}>Aprobar</button>{/if}
					{#if selected.status === 'approved'}<button class="btn" on:click={() => move('publish')} disabled={busy}>Publicar</button>{/if}
					{#if selected.status === 'published'}<a class="btn" href={url(selected)}>Ver en el blog</a>{/if}
				</div>
			</div>
		</section>
	{/if}

	<div class="feed">
		{#each $articles as a}
			<button class="post" style="text-align:left;cursor:pointer;font:inherit" on:click={() => open(a.id)}>
				{#if a.has_image}<img class="thumb" src={`/api/articles/${a.id}/image`} alt="" loading="lazy" />{:else}<div class="thumb empty">Sin imagen</div>{/if}
				<div><h3>{a.title}</h3><p>{a.summary}</p><p style="margin-top:8px"><span class="chip {a.status}">{statusName[a.status]}</span> <span class="muted small">{a.published_at ? new Date(a.published_at).toLocaleDateString('es-MX', { dateStyle: 'medium' }) : ''}</span></p></div>
			</button>
		{:else}
			<div class="panel"><p class="muted" style="margin:0">Aún no hay artículos.</p></div>
		{/each}
	</div>
</div>
