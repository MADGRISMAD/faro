<script lang="ts">
	import { api } from '#lib/api';
	import { research, siteId, refreshSite } from '#lib/app';

	let busy = false, result: { connected: boolean; reason: string } | null = null, copied = '', recipe = '';
	const names: Record<string, string> = { wordpress: 'WordPress', wix: 'Wix', shopify: 'Shopify', squarespace: 'Squarespace', webflow: 'Webflow', vercel: 'Vercel', netlify: 'Netlify', cloudflare: 'Cloudflare', nginx: 'Nginx', apache: 'Apache', caddy: 'Caddy', other: '' };
	// Plataformas donde no se puede enviar una ruta a otro servidor: ahí se usa el enlace o el widget.
	const closed = ['wordpress', 'wix', 'shopify', 'squarespace', 'webflow', 'vercel', 'netlify'];
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
		apache: `# Necesita mod_proxy, mod_proxy_http, mod_ssl y mod_headers\nSSLProxyEngine on\nProxyPreserveHost Off\nRequestHeader set x-faro-domain "${domain}"\nProxyPass ${path} https://${faro}${path}\nProxyPassReverse ${path} https://${faro}${path}`
	} as Record<string, string>;
	$: if (!recipe) recipe = ['nginx', 'caddy', 'cloudflare', 'apache'].includes(platform) ? platform : 'nginx';
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

<div class="view">
	<h1>Conectar mi sitio</h1>
	<p class="sub">Tu blog ya está publicado. Aquí eliges cómo llega a {domain}.</p>

	{#if r?.connected}
		<div class="panel" style="margin-bottom:18px;border-color:#1b603b"><b>Conectado.</b> <span class="muted">Tus artículos salen en <a href={r.blog_base}>{r.blog_base}</a>.</span></div>
	{/if}

	<section class="panel" style="margin-bottom:18px">
		<div class="row"><h2 style="font-size:1.25rem;margin:0">1 · Tu blog ya existe <span class="chip published">Listo</span></h2></div>
		<p class="muted">Está publicado en una dirección de FaroContenido. No necesitas instalar nada: agrega un enlace «Blog» en el menú de tu sitio que apunte aquí.</p>
		<div class="code">{r?.hosted_url}</div>
		<div style="display:flex;gap:10px;margin-top:10px;flex-wrap:wrap">
			<a class="btn small" href={r?.hosted_url} target="_blank" rel="noopener">Ver mi blog</a>
			<button class="btn secondary small" on:click={() => copy('hosted', r?.hosted_url || '')}>{copied === 'hosted' ? 'Copiado' : 'Copiar dirección'}</button>
		</div>
	</section>

	<section class="panel" style="margin-bottom:18px">
		<h2 style="font-size:1.25rem;margin:0 0 4px">2 · En tu propio dominio <span class="chip">Mejor para SEO</span></h2>
		<p class="muted" style="margin-top:0">Así tus artículos viven en <b>{r?.canonical_url}{r?.blog_prefix}</b> y el prestigio de Google se queda en tu sitio. Tu servidor debe enviar esa ruta a FaroContenido.</p>
		{#if names[platform]}
			<p class="panel" style="background:#faf8f2;margin:0 0 14px">Detectamos que tu sitio usa <b>{names[platform]}</b>.
				{#if closed.includes(platform)}
					En {names[platform]} no se puede enviar una ruta a otro servidor. Usa la opción 1 (enlace en tu menú) o la 3 (widget), o pon Cloudflare delante de tu dominio y usa su receta de abajo.
				{:else}
					Te mostramos primero la receta que le corresponde.
				{/if}
			</p>
		{/if}
		<div style="display:flex;gap:8px;flex-wrap:wrap;margin-bottom:12px">
			{#each [['nginx', 'Nginx'], ['caddy', 'Caddy'], ['cloudflare', 'Cloudflare'], ['apache', 'Apache']] as [id, label]}
				<button class="btn small" class:secondary={recipe !== id} on:click={() => (recipe = id)}>{label}</button>
			{/each}
		</div>
		<div class="code">{recipes[recipe]}</div>
		<div style="display:flex;gap:10px;margin-top:10px;flex-wrap:wrap;align-items:center">
			<button class="btn secondary small" on:click={() => copy('recipe', recipes[recipe])}>{copied === 'recipe' ? 'Copiado' : 'Copiar receta'}</button>
			<button class="btn small" on:click={verify} disabled={busy}>{busy ? 'Comprobando…' : 'Verificar conexión'}</button>
		</div>
		{#if result}
			<p class={result.connected ? 'success' : 'error'} role="status" style="margin-top:12px">{result.connected ? 'Conectado: tu sitio ya sirve el blog.' : result.reason}</p>
		{/if}
		<p class="muted small" style="margin-bottom:0">¿Tu blog vive en otra ruta o subdominio? Cámbialo en <a href="/dashboard/ajustes">Ajustes</a>.</p>
	</section>

	<section class="panel">
		<h2 style="font-size:1.25rem;margin:0 0 4px">3 · Lista de artículos dentro de una página tuya</h2>
		<p class="muted" style="margin-top:0">Pega este código en una página de tu sitio (en WordPress, Wix, Shopify o Webflow, en un bloque «HTML personalizado»). Muestra tus artículos; al abrirlos se leen en {r?.connected ? 'tu dominio' : 'tu blog alojado, hasta que conectes tu dominio'}.</p>
		<div class="code">{widget}</div>
		<button class="btn secondary small" style="margin-top:10px" on:click={() => copy('widget', widget)}>{copied === 'widget' ? 'Copiado' : 'Copiar código'}</button>
	</section>
</div>
