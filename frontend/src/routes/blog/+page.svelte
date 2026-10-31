<script lang="ts">
	import { canonical } from '#lib/seo';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	export let data;
</script>
<svelte:head><meta name="faro-site" content={data.site.id} /><title>Blog | {data.site.name}</title><meta name="description" content={data.site.brand_description||`Artículos de ${data.site.name}`} /><link rel="canonical" href={canonical(data.site)} /><meta property="og:title" content={`Blog | ${data.site.name}`} /><meta property="og:type" content="website" /><meta property="og:url" content={canonical(data.site)} /></svelte:head>
<main class="mx-auto flex w-full max-w-5xl flex-col gap-8 px-4 py-10">
	<div class="flex flex-col gap-3">
		<p class="text-sm text-muted-foreground"><a class="underline-offset-4 hover:underline" href={data.site.canonical_url}>← {data.site.name}</a> · Blog</p>
		<h1 class="text-4xl font-semibold tracking-tight">Ideas para seguir adelante.</h1>
		<p class="text-muted-foreground">{data.site.brand_description}</p>
	</div>
	{#if data.articles.length === 0}
		<Card.Root><Card.Header><Card.Title><h2>Aún no hay artículos</h2></Card.Title><Card.Description>Muy pronto habrá contenido nuevo.</Card.Description></Card.Header></Card.Root>
	{:else}
		<div class="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
			{#each data.articles as article}
				<Card.Root>
					{#if article.has_image}<a class="px-4" href="{data.site.blog_prefix}/{article.slug}"><img src="{data.site.blog_prefix}/{article.slug}/image" alt="" loading="lazy" width="1376" height="768" class="aspect-video w-full rounded-lg object-cover" /></a>{/if}
					<Card.Header>
						<Badge variant="secondary" class="w-fit">{new Date(article.published_at).toLocaleDateString('es-MX')}</Badge>
						<Card.Title class="text-lg"><h2><a href="{data.site.blog_prefix}/{article.slug}">{article.title}</a></h2></Card.Title>
						<Card.Description>{article.summary}</Card.Description>
					</Card.Header>
					<Card.Footer><a class="text-sm underline-offset-4 hover:underline" href="{data.site.blog_prefix}/{article.slug}">Leer artículo →</a></Card.Footer>
				</Card.Root>
			{/each}
		</div>
	{/if}
</main>
