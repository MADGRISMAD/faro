<script lang="ts">
	import { canonical } from '#lib/seo';
	export let data;
	const a = data.article, site = data.site;
	const url = canonical(site, a.slug);
	// La imagen siempre se pide a Faro con dirección absoluta: esta página también se sirve desde el dominio del cliente (/x), donde una ruta relativa no existe.
	const base = (data.base || 'https://faro.hyperionmkt.com').replace(/\/$/, '');
	const img = a.has_image ? `${base}/api/public/sites/${site.id}/articles/${encodeURIComponent(a.slug)}/image` : '';
	const origin = site.canonical_url.replace(/\/$/, '');
	const graph: Record<string, unknown>[] = [
		{
			'@type': 'BlogPosting', headline: a.title, description: a.meta_description, datePublished: a.published_at, dateModified: a.published_at, inLanguage: 'es-MX',
			mainEntityOfPage: url, ...(img ? { image: img } : {}),
			author: { '@type': 'Organization', name: site.name, url: origin }, publisher: { '@type': 'Organization', name: site.name, url: origin }
		},
		{
			'@type': 'BreadcrumbList',
			itemListElement: [
				{ '@type': 'ListItem', position: 1, name: 'Inicio', item: origin + '/' },
				{ '@type': 'ListItem', position: 2, name: 'Blog', item: origin + site.blog_prefix },
				{ '@type': 'ListItem', position: 3, name: a.title, item: url }
			]
		}
	];
	if (a.content.faq.length) graph.push({ '@type': 'FAQPage', mainEntity: a.content.faq.map((f: { question: string; answer: string }) => ({ '@type': 'Question', name: f.question, acceptedAnswer: { '@type': 'Answer', text: f.answer } })) });
	// Dentro de <script> Svelte no interpola: el bloque entero se inserta como HTML ("<" escapado para que el JSON no cierre la etiqueta).
	const ld = `<script type="application/ld+json">${JSON.stringify({ '@context': 'https://schema.org', '@graph': graph }).replace(/</g, '\\u003c')}</` + 'script>';
</script>
<svelte:head><meta name="faro-site" content={data.site.id} /><title>{data.article.title} | {data.site.name}</title><meta name="description" content={data.article.meta_description} /><link rel="canonical" href={url} /><meta property="og:type" content="article" /><meta property="og:title" content={data.article.title} /><meta property="og:description" content={data.article.meta_description} /><meta property="og:url" content={url} />{#if img}<meta property="og:image" content={img} /><meta name="twitter:card" content="summary_large_image" />{/if}<!-- eslint-disable-next-line svelte/no-at-html-tags -->
{@html ld}</svelte:head>
<main><article class="article-body"><p class="eyebrow"><a href={data.site.blog_prefix}>← Blog de {data.site.name}</a></p><h1>{data.article.title}</h1><p class="muted">Publicado el {new Date(data.article.published_at).toLocaleDateString('es-MX')}</p>{#if img}<img src={img} alt="Portada: {data.article.title}" width="1376" height="768" style="width:100%;height:auto;border-radius:12px;margin:8px 0 20px" />{/if}<p class="lead">{data.article.summary}</p><p>{data.article.content.intro}</p>{#each data.article.content.sections as section}<section>{#if section.level === 3}<h3>{section.heading}</h3>{:else}<h2>{section.heading}</h2>{/if}{#each section.paragraphs as paragraph}<p>{paragraph}</p>{/each}{#if section.bullets.length}<ul>{#each section.bullets as bullet}<li>{bullet}</li>{/each}</ul>{/if}</section>{/each}{#if data.article.content.faq.length}<section><h2>Preguntas frecuentes</h2>{#each data.article.content.faq as item}<h3>{item.question}</h3><p>{item.answer}</p>{/each}</section>{/if}</article></main>
