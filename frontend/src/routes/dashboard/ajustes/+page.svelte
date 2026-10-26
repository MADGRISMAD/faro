<script lang="ts">
	import { api } from '#lib/api';
	import { sites, siteId, research, user, refreshSite, loadSites } from '#lib/app';

	let copied = false, error = '', notice = '', canonical = '', prefix = '/blog', loaded = '';
	$: if (site && loaded !== site.id) {
		loaded = site.id;
		canonical = site.canonical_url;
		prefix = site.blog_prefix;
	}
	async function saveAddress() {
		if (!site) return;
		error = notice = '';
		try {
			await api(`/sites/${site.id}`, { method: 'PATCH', body: JSON.stringify({ name: site.name, domain: site.domain, canonical_url: canonical, blog_prefix: prefix, brand_description: site.brand_description, tone: site.tone }) });
			notice = 'Dirección guardada.';
			await loadSites();
		} catch (e) {
			error = (e as Error).message;
		}
	}
	$: site = $sites.find((s) => s.id === $siteId);
	$: origin = typeof location === 'undefined' ? '' : location.origin;
	$: code = `<div id="faro-blog"></div>\n<script src="${origin}/api/embed/${$siteId}.js" defer><\/script>`;
	$: left = $user ? ($user.budget_microusd - $user.spent_microusd - $user.reserved_microusd) / 1e6 : 0;
	async function copy() {
		try {
			await navigator.clipboard.writeText(code);
			copied = true;
			setTimeout(() => (copied = false), 1500);
		} catch {}
	}
	async function perWeek(n: number) {
		error = '';
		try {
			await api(`/sites/${$siteId}/autopilot`, { method: 'PATCH', body: JSON.stringify({ enabled: $research?.autopilot ?? true, posts_per_week: n }) });
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		}
	}
</script>

<div class="view">
	<h1>Ajustes</h1>
	<p class="sub">Integraciones y preferencias de {site?.domain}</p>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if notice}<p class="success" role="status">{notice}</p>{/if}

	<section class="panel" style="margin-bottom:18px">
		<h2 style="font-size:1.25rem;margin:0 0 4px">Conecta tu sitio</h2>
		<p class="muted" style="margin-top:0">Pega este código donde quieras que aparezca tu blog. El SEO real lo da el blog publicado en <b>{site?.canonical_url}{site?.blog_prefix}</b>.</p>
		<div class="code">{code}</div>
		<button class="btn secondary small" style="margin-top:10px" on:click={copy}>{copied ? 'Copiado' : 'Copiar código'}</button>
	</section>

	<section class="panel" style="margin-bottom:18px">
		<h2 style="font-size:1.25rem;margin:0 0 4px">Dirección del blog</h2>
		<p class="muted" style="margin-top:0">Si tu blog vive en otra ruta o subdominio, ajústalo aquí. Se usa en los enlaces, el sitemap y los datos para Google.</p>
		<form class="stack" on:submit|preventDefault={saveAddress}>
			<label>URL de tu sitio<input bind:value={canonical} required /></label>
			<label>Ruta del blog<input bind:value={prefix} required /></label>
			<div><button class="btn secondary small">Guardar dirección</button></div>
		</form>
	</section>

	<section class="panel" style="margin-bottom:18px">
		<h2 style="font-size:1.25rem;margin:0 0 4px">Ritmo de publicación</h2>
		<p class="muted" style="margin-top:0">Las primeras dos semanas sale 1 por semana y nunca más de 3: un ritmo constante y moderado le sienta mejor a Google que publicar mucho de golpe.</p>
		<div style="display:flex;gap:10px">
			{#each [1, 2, 3] as n}<button class="btn small" class:secondary={$research?.posts_per_week !== n} on:click={() => perWeek(n)}>{n} por semana</button>{/each}
		</div>
	</section>

	<section class="panel">
		<h2 style="font-size:1.25rem;margin:0 0 4px">Uso de IA</h2>
		<p class="muted" style="margin:0">Crédito disponible: <b>US${left.toFixed(2)}</b> · consumido: US${($user?.spent_microusd / 1e6 || 0).toFixed(3)}. Cada artículo con su imagen cuesta alrededor de US$0.05.</p>
	</section>
</div>
