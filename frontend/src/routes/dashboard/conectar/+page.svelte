<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { api } from '#lib/api';
	import { research, siteId, refreshSite } from '#lib/app';

	let busy = false, result: { connected: boolean; reason: string } | null = null, copied = '', recipe = '';
	const names: Record<string, string> = { wordpress: 'WordPress', wix: 'Wix', shopify: 'Shopify', squarespace: 'Squarespace', webflow: 'Webflow', vercel: 'Vercel', netlify: 'Netlify', cloudflare: 'Cloudflare', nginx: 'Nginx', apache: 'Apache', caddy: 'Caddy', other: '' };
	// Plataformas donde no se puede enviar una ruta a otro servidor: ahí se usa el enlace o el widget.
	const closed = ['wordpress', 'wix', 'shopify', 'squarespace', 'webflow'];
	$: r = $research;
	$: platform = r?.research?.platform || 'other';
	$: faro = (() => {
		try { return new URL(r?.hosted_url || '').host; } catch { return location.host; }
	})();
	$: domain = r?.domain || '';
	$: path = r?.blog_prefix || '/blog';
	$: recipes = {
		nginx: `# En el bloque server de tu dominio\nlocation ~ ^${path}(/|$) {\n    proxy_pass https://${faro};\n    proxy_ssl_server_name on;\n    proxy_set_header Host ${faro};\n    proxy_set_header x-faro-domain ${domain};\n}`,
		caddy: `${domain} {\n    handle ${path}* {\n        reverse_proxy https://${faro} {\n            header_up Host ${faro}\n            header_up x-faro-domain ${domain}\n        }\n    }\n    # ...el resto de tu sitio\n}`,
		cloudflare: `// Cloudflare → Workers → Crear Worker, pega esto y agrega la ruta ${domain}${path}*\nexport default {\n  fetch(request) {\n    const url = new URL(request.url);\n    url.hostname = '${faro}';\n    const req = new Request(url, request);\n    req.headers.set('x-faro-domain', '${domain}');\n    return fetch(req);\n  }\n}`,
		nextjs: `// next.config.mjs (sitio en Vercel con Next.js). Después haz deploy.\nconst nextConfig = {\n  async rewrites() {\n    return [\n      { source: '${path}', destination: 'https://${faro}/x/${domain}' },\n      { source: '${path}/:path*', destination: 'https://${faro}/x/${domain}/:path*' }\n    ];\n  }\n};\nexport default nextConfig;`,
		netlify: `# Archivo _redirects en la carpeta publicada de tu sitio (Netlify)\n${path}        https://${faro}/x/${domain}          200!\n${path}/*      https://${faro}/x/${domain}/:splat   200!`,
		apache: `# Necesita mod_proxy, mod_proxy_http, mod_ssl y mod_headers\nSSLProxyEngine on\nProxyPreserveHost Off\nRequestHeader set x-faro-domain "${domain}"\nProxyPass ${path} https://${faro}${path}\nProxyPassReverse ${path} https://${faro}${path}`
	} as Record<string, string>;
	$: if (!recipe) recipe = platform === 'vercel' ? 'nextjs' : ['nginx', 'caddy', 'cloudflare', 'apache', 'netlify'].includes(platform) ? platform : 'nginx';
	// El ejemplo se arma con líneas sueltas para no mezclar las comillas invertidas del código de ejemplo con las de este archivo.
	const bt = '`', dl = '$';
	$: nextLib = [
		'// lib/faro.js: Faro solo entrega los datos; tú los pintas con tu diseño.',
		"const FARO = 'https://" + faro + "';",
		"const SITE = '" + domain + "';",
		'',
		'export async function getPosts() {',
		'  const r = await fetch(' + bt + dl + '{FARO}/api/v1/' + dl + '{SITE}/posts' + bt + ', { next: { revalidate: 300 } });',
		'  return r.ok ? r.json() : { site: null, posts: [] };',
		'}',
		'',
		'export async function getPost(slug) {',
		'  const r = await fetch(' + bt + dl + '{FARO}/api/v1/' + dl + '{SITE}/posts/' + dl + '{slug}' + bt + ', { next: { revalidate: 300 } });',
		'  return r.ok ? r.json() : null;',
		'}'
	].join('\n');
	$: nextPage = [
		'// app' + path + '/[slug]/page.js',
		"import { notFound } from 'next/navigation';",
		"import { getPost } from '@/lib/faro';",
		'',
		'export async function generateMetadata({ params }) {',
		'  const post = await getPost((await params).slug);',
		'  if (!post) return {};',
		'  return {',
		'    title: post.seo.title,',
		'    description: post.seo.description,',
		'    alternates: { canonical: post.canonical },',
		'    openGraph: { images: post.image ? [post.image.url] : [] },',
		"    other: { 'faro-site': post.site_id } // para que Faro marque tu sitio como conectado",
		'  };',
		'}',
		'',
		'export default async function Page({ params }) {',
		'  const post = await getPost((await params).slug);',
		'  if (!post) notFound();',
		'  return (',
		'    <article>',
		'      {post.jsonld.map((d, i) => <script key={i} type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(d) }} />)}',
		'      <h1>{post.title}</h1>',
		'      {post.image && <img src={post.image.url} alt={post.image.alt} />}',
		'      <div dangerouslySetInnerHTML={{ __html: post.html }} />',
		'    </article>',
		'  );',
		'}'
	].join('\n');
	$: widget = `<div id="faro-blog"></div>\n<script src="${typeof location === 'undefined' ? '' : location.origin}/api/embed/${$siteId}.js" defer><\/script>`;

	async function copy(key: string, text: string) {
		try {
			await navigator.clipboard.writeText(text);
			copied = key;
			setTimeout(() => (copied = ''), 1400);
		} catch {}
	}
	async function verify() {
		busy = true;
		result = null;
		try {
			result = await api(`/sites/${$siteId}/verify`, { method: 'POST' });
			await refreshSite();
		} catch (e) {
			result = { connected: false, reason: (e as Error).message };
		} finally {
			busy = false;
		}
	}
</script>

<div>
	<h1 class="text-2xl font-semibold tracking-tight">Conectar mi sitio</h1>
	<p class="text-sm text-muted-foreground">Tu blog ya está publicado. Aquí eliges cómo llega a {domain}.</p>
</div>

{#if r?.connected}
	<Alert.Root role="status"><Alert.Title>Conectado.</Alert.Title><Alert.Description>Tus artículos salen en <a class="underline" href={r.blog_base}>{r.blog_base}</a>.</Alert.Description></Alert.Root>
{/if}

<Card.Root>
	<Card.Header>
		<Card.Title class="flex items-center gap-2">1 · Tu blog ya existe <Badge>Listo</Badge></Card.Title>
		<Card.Description>Está publicado en una dirección de Faro. No necesitas instalar nada: agrega un enlace «Blog» en el menú de tu sitio que apunte aquí.</Card.Description>
	</Card.Header>
	<Card.Content><pre class="overflow-x-auto rounded-lg bg-muted p-4 font-mono text-sm">{r?.hosted_url}</pre></Card.Content>
	<Card.Footer class="flex-wrap gap-2">
		<Button size="sm" href={r?.hosted_url} target="_blank" rel="noopener">Ver mi blog</Button>
		<Button variant="secondary" size="sm" onclick={() => copy('hosted', r?.hosted_url || '')}>{copied === 'hosted' ? 'Copiado' : 'Copiar dirección'}</Button>
	</Card.Footer>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title class="flex items-center gap-2">2 · Con tu propio diseño <Badge variant="secondary">Recomendado</Badge></Card.Title>
		<Card.Description>Faro solo entrega los datos (texto, imagen y SEO ya listos) y tu sitio los pinta como quiera. Es la mejor opción si ya tienes diseño: tus artículos viven en <b class="text-foreground">{r?.canonical_url}{r?.blog_prefix}</b> con tu estilo.</Card.Description>
	</Card.Header>
	<Card.Content class="flex flex-col gap-4">
		<pre class="overflow-x-auto rounded-lg bg-muted p-4 font-mono text-sm">GET https://{faro}/api/v1/{domain}/posts
GET https://{faro}/api/v1/{domain}/posts/{'{slug}'}</pre>
		<p class="text-sm text-muted-foreground">Cada artículo trae <code>title</code>, <code>summary</code>, <code>image</code> (url, alt, tamaño), <code>content</code> estructurado, <code>html</code> listo para insertar, <code>seo</code>, <code>canonical</code> y <code>jsonld</code>. Es contenido público con CORS abierto.</p>
		<h3 class="font-medium">Ejemplo para Next.js</h3>
		<pre class="overflow-x-auto rounded-lg bg-muted p-4 font-mono text-sm">{nextLib}</pre>
		<div><Button variant="secondary" size="sm" onclick={() => copy('lib', nextLib)}>{copied === 'lib' ? 'Copiado' : 'Copiar lib/faro.js'}</Button></div>
		<pre class="overflow-x-auto rounded-lg bg-muted p-4 font-mono text-sm">{nextPage}</pre>
		{#if result}<Alert.Root variant={result.connected ? 'default' : 'destructive'} role="status"><Alert.Description>{result.connected ? 'Conectado: tu sitio ya sirve el blog.' : result.reason}</Alert.Description></Alert.Root>{/if}
	</Card.Content>
	<Card.Footer class="flex-wrap gap-2">
		<Button variant="secondary" size="sm" onclick={() => copy('page', nextPage)}>{copied === 'page' ? 'Copiado' : 'Copiar página del artículo'}</Button>
		<Button size="sm" onclick={verify} disabled={busy}>{busy ? 'Comprobando…' : 'Verificar conexión'}</Button>
	</Card.Footer>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title class="flex items-center gap-2">3 · Faro sirve la página en tu dominio <Badge variant="secondary">Sin escribir código de página</Badge></Card.Title>
		<Card.Description>Así tus artículos viven en <b class="text-foreground">{r?.canonical_url}{r?.blog_prefix}</b> y el prestigio de Google se queda en tu sitio. Tu servidor debe enviar esa ruta a Faro.</Card.Description>
	</Card.Header>
	<Card.Content class="flex flex-col gap-4">
		{#if names[platform]}
			<Alert.Root>
				<Alert.Description>Detectamos que tu sitio usa <b>{names[platform]}</b>.
					{#if closed.includes(platform)}
						En {names[platform]} no se puede enviar una ruta a otro servidor. Usa la opción 1 (enlace en tu menú) o la 3 (widget), o pon Cloudflare delante de tu dominio y usa su receta de abajo.
					{:else}
						Te mostramos primero la receta que le corresponde.
					{/if}
				</Alert.Description>
			</Alert.Root>
		{/if}
		<Tabs.Root bind:value={recipe}>
			<Tabs.List class="h-auto flex-wrap">
				{#each [['nextjs', 'Next.js / Vercel'], ['netlify', 'Netlify'], ['nginx', 'Nginx'], ['caddy', 'Caddy'], ['cloudflare', 'Cloudflare'], ['apache', 'Apache']] as [id, label]}
					<Tabs.Trigger value={id}>{label}</Tabs.Trigger>
				{/each}
			</Tabs.List>
		</Tabs.Root>
		<pre class="overflow-x-auto rounded-lg bg-muted p-4 font-mono text-sm">{recipes[recipe]}</pre>
		{#if result}<Alert.Root variant={result.connected ? 'default' : 'destructive'} role="status"><Alert.Description>{result.connected ? 'Conectado: tu sitio ya sirve el blog.' : result.reason}</Alert.Description></Alert.Root>{/if}
		<p class="text-sm text-muted-foreground">¿Tu blog vive en otra ruta o subdominio? Cámbialo en <a class="underline" href="/dashboard/ajustes">Ajustes</a>.</p>
	</Card.Content>
	<Card.Footer class="flex-wrap gap-2">
		<Button variant="secondary" size="sm" onclick={() => copy('recipe', recipes[recipe])}>{copied === 'recipe' ? 'Copiado' : 'Copiar receta'}</Button>
		<Button size="sm" onclick={verify} disabled={busy}>{busy ? 'Comprobando…' : 'Verificar conexión'}</Button>
	</Card.Footer>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title>4 · Lista de artículos dentro de una página tuya</Card.Title>
		<Card.Description>Pega este código en una página de tu sitio (en WordPress, Wix, Shopify o Webflow, en un bloque «HTML personalizado»). Muestra tus artículos; al abrirlos se leen en {r?.connected ? 'tu dominio' : 'tu blog alojado, hasta que conectes tu dominio'}.</Card.Description>
	</Card.Header>
	<Card.Content><pre class="overflow-x-auto rounded-lg bg-muted p-4 font-mono text-sm">{widget}</pre></Card.Content>
	<Card.Footer><Button variant="secondary" size="sm" onclick={() => copy('widget', widget)}>{copied === 'widget' ? 'Copiado' : 'Copiar código'}</Button></Card.Footer>
</Card.Root>
