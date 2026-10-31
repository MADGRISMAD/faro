<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import * as Item from '$lib/components/ui/item/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
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

<div class="flex flex-wrap items-start justify-between gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Artículos</h1>
		<p class="text-sm text-muted-foreground">Administra tu contenido SEO</p>
	</div>
	<Button variant="outline" onclick={() => (showNew = !showNew)}>+ Nuevo artículo</Button>
</div>
{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}
{#if notice}<Alert.Root role="status"><Alert.Description>{notice}</Alert.Description></Alert.Root>{/if}

{#if showNew}
	<Card.Root>
		<Card.Content>
			<form class="flex flex-wrap gap-2" onsubmit={(e) => { e.preventDefault(); (create)(); }}>
				<Input bind:value={topic} required maxlength={300} placeholder="¿De qué quieres escribir?" class="min-w-60 flex-1" />
				<Button type="submit" disabled={busy}>Escribir con IA</Button>
			</form>
		</Card.Content>
	</Card.Root>
{/if}

<div class="grid gap-4 sm:grid-cols-3">
	<Card.Root><Card.Header><Card.Description>Total</Card.Description><Card.Title class="text-3xl">{$articles.length}</Card.Title></Card.Header></Card.Root>
	<Card.Root><Card.Header><Card.Description>Borradores</Card.Description><Card.Title class="text-3xl">{$articles.length - published}</Card.Title></Card.Header></Card.Root>
	<Card.Root><Card.Header><Card.Description>Publicados</Card.Description><Card.Title class="text-3xl">{published}</Card.Title></Card.Header></Card.Root>
</div>

{#if selected}
	<Card.Root>
		<Card.Header>
			<Card.Title class="text-xl"><h2>{selected.title}</h2></Card.Title>
			<Card.Action><Button variant="secondary" size="sm" onclick={() => (selected = null)}>Cerrar</Button></Card.Action>
		</Card.Header>
		<Card.Content class="flex flex-col gap-4">
			{#if Array.isArray(selected.review_notes) && selected.review_notes.length}
				<Alert.Root role="note">
					<Alert.Title>Revisa esto antes de publicar</Alert.Title>
					<Alert.Description>
						<ul class="list-disc pl-5">{#each selected.review_notes as n}<li>{n}</li>{/each}</ul>
						<p class="mt-2">Faro no publica solo un artículo con estas notas. Al guardar tus cambios se vuelve a revisar.</p>
					</Alert.Description>
				</Alert.Root>
			{/if}
			{#if selected.has_image}<img class="aspect-video w-full rounded-lg object-cover" src={`/api/articles/${selected.id}/image`} alt="Portada del artículo" />{/if}
			<Label class="flex-col items-start">Título<Input bind:value={selected.title} disabled={lock()} /></Label>
			<Label class="flex-col items-start">Slug<Input bind:value={selected.slug} disabled={lock()} /></Label>
			<Label class="flex-col items-start">Meta description<Textarea bind:value={selected.meta_description} disabled={lock()} /></Label>
			<Label class="flex-col items-start">Resumen<Textarea bind:value={selected.summary} disabled={lock()} /></Label>
			<Label class="flex-col items-start">Introducción<Textarea bind:value={selected.content.intro} disabled={lock()} /></Label>
			{#each selected.content.sections as section}
				<div class="flex flex-col gap-3 rounded-lg border p-4">
					<Label class="flex-col items-start">Encabezado<Input bind:value={section.heading} disabled={lock()} /></Label>
					{#each section.paragraphs as _, i}<Label class="flex-col items-start">Párrafo<Textarea bind:value={section.paragraphs[i]} disabled={lock()} /></Label>{/each}
					{#each section.bullets as _, i}<Label class="flex-col items-start">Viñeta<Input bind:value={section.bullets[i]} disabled={lock()} /></Label>{/each}
				</div>
			{/each}
			{#each selected.content.faq as faq}
				<div class="flex flex-col gap-3 rounded-lg border p-4">
					<Label class="flex-col items-start">Pregunta<Input bind:value={faq.question} disabled={lock()} /></Label>
					<Label class="flex-col items-start">Respuesta<Textarea bind:value={faq.answer} disabled={lock()} /></Label>
				</div>
			{/each}
		</Card.Content>
		<Card.Footer class="flex-wrap gap-2">
			{#if selected.status !== 'published'}<Button variant="secondary" onclick={save} disabled={busy}>Guardar cambios</Button>{/if}
			{#if selected.status === 'draft'}<Button onclick={() => move('approve')} disabled={busy}>Aprobar</Button>{/if}
			{#if selected.status === 'approved'}<Button onclick={() => move('publish')} disabled={busy}>Publicar</Button>{/if}
			{#if selected.status === 'published'}<Button href={url(selected)}>Ver en el blog</Button>{/if}
		</Card.Footer>
	</Card.Root>
{/if}

<Item.Group class="gap-3">
	{#each $articles as a}
		<Item.Root variant="outline" class="hover:bg-muted/50">
			{#snippet child({ props })}
				<button {...props} type="button" class="{props.class} w-full cursor-pointer text-left" onclick={() => open(a.id)}>
					<Item.Media variant="image" class="h-16 w-28">
						{#if a.has_image}<img src={`/api/articles/${a.id}/image`} alt="" loading="lazy" />{:else}<span class="grid size-full place-items-center bg-muted text-xs text-muted-foreground">Sin imagen</span>{/if}
					</Item.Media>
					<Item.Content>
						<Item.Title>{a.title}</Item.Title>
						<Item.Description>{a.summary}</Item.Description>
						<span class="flex flex-wrap items-center gap-2">
							<Badge variant={a.status === 'published' ? 'default' : 'secondary'}>{statusName[a.status]}</Badge>
							{#if typeof a.review_notes === 'number' && a.review_notes > 0}<Badge variant="outline">Por revisar</Badge>{/if}
							<span class="text-xs text-muted-foreground">{a.published_at ? new Date(a.published_at).toLocaleDateString('es-MX', { dateStyle: 'medium' }) : ''}</span>
						</span>
					</Item.Content>
				</button>
			{/snippet}
		</Item.Root>
	{:else}
		<Card.Root><Card.Content class="text-sm text-muted-foreground">Aún no hay artículos.</Card.Content></Card.Root>
	{/each}
</Item.Group>
