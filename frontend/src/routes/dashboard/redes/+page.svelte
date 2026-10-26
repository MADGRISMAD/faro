<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '#lib/api';
	import { siteId, sites } from '#lib/app';

	type Post = { title: string; status: string; url: string; error: string; at: string };
	type Data = { configured: boolean; status: 'none' | 'choosing' | 'connected' | 'broken'; page_name?: string; error?: string; choices: { id: string; name: string }[]; posts: Post[] };
	let data: Data | null = null, error = '', notice = '', busy = false, pick = '', loadedFor = '';
	const notes: Record<string, string> = {
		ok: 'Facebook conectado. Desde ahora, cada artículo que se publique se compartirá en tu página.',
		elegir: 'Elige en qué página quieres que se compartan tus artículos.',
		denied: 'No diste permiso en Facebook, así que no se conectó nada.',
		nopages: 'Esa cuenta de Facebook no administra ninguna página donde pueda publicar. Revisa que seas administrador de la página.',
		error: 'No se pudo conectar Facebook. Inténtalo de nuevo.'
	};
	const statusName: Record<string, string> = { pending: 'En cola', sending: 'Enviando', sent: 'Publicado', failed: 'No se publicó', unknown: 'Revisa tu página', cancelled: 'Cancelado' };
	$: domain = $sites.find((s) => s.id === $siteId)?.domain || '';

	async function load() {
		const id = $siteId;
		error = '';
		try {
			const r = await api(`/sites/${id}/facebook`);
			if (id === $siteId) data = r; // si cambiaste de sitio mientras cargaba, se ignora la respuesta vieja
		} catch (e) {
			error = (e as Error).message;
		}
	}
	onMount(() => {
		const f = new URLSearchParams(location.search).get('facebook');
		if (f) notice = notes[f] || '';
		// se recarga al cambiar de sitio en la barra lateral
		return siteId.subscribe((id) => {
			if (!id || id === loadedFor) return;
			loadedFor = id;
			data = null;
			load();
		});
	});
	async function run(fn: () => Promise<void>) {
		busy = true;
		error = '';
		try {
			await fn();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	const connect = () => run(async () => {
		const r = await api(`/sites/${$siteId}/facebook/start`, { method: 'POST' });
		location.href = r.url;
	});
	const choose = () => run(async () => {
		await api(`/sites/${$siteId}/facebook/page`, { method: 'POST', body: JSON.stringify({ page_id: pick }) });
		notice = notes.ok;
		await load();
	});
	const disconnect = () => run(async () => {
		if (!confirm('Faro dejará de compartir tus artículos en Facebook. Lo que ya se publicó en tu página no se borra. ¿Desconectar?')) return;
		await api(`/sites/${$siteId}/facebook`, { method: 'DELETE' });
		notice = '';
		await load();
	});
	const when = (s: string) => new Date(s).toLocaleDateString('es-MX', { day: 'numeric', month: 'short', hour: 'numeric', minute: '2-digit' });
</script>

<div class="view redes">
	<h1>Redes sociales</h1>
	<p class="sub">Cada artículo que Faro publique en {domain || 'tu sitio'} se comparte solo en tus redes, con su portada y el enlace al artículo.</p>
	{#if notice}<p class="success" role="status">{notice}</p>{/if}
	{#if error}<p class="error" role="alert">{error}</p>{/if}

	<section class="panel net">
		<div class="net-head">
			<span class="logo fb" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M14 8.5V6.8c0-.8.5-1 1-1h2V2.2h-2.9C11 2.2 10.4 4.5 10.4 6v2.5H8v3.7h2.4V22H14v-9.8h2.7l.4-3.7H14Z" /></svg></span>
			<div class="grow">
				<h2>Facebook</h2>
				{#if !data}<small>Cargando…</small>
				{:else if data.status === 'connected'}<small>Conectado a <b>{data.page_name}</b></small>
				{:else if data.status === 'broken'}<small class="bad">La conexión dejó de funcionar</small>
				{:else}<small>Comparte cada artículo en la página de tu negocio</small>{/if}
			</div>
			{#if data?.status === 'connected'}<span class="chip ok-chip"><i></i>Activo</span>{/if}
		</div>

		{#if data && !data.configured}
			<p class="muted">La conexión con Facebook se está terminando de configurar en el servidor. Muy pronto podrás usarla.</p>
		{:else if data?.status === 'choosing' && data.choices.length}
			<fieldset class="pages">
				<legend>¿En qué página publicamos?</legend>
				{#each data.choices as c}
					<label class="page" class:on={pick === c.id}><input type="radio" name="page" value={c.id} bind:group={pick} />{c.name}</label>
				{/each}
			</fieldset>
			<div class="acts"><button class="btn" disabled={!pick || busy} on:click={choose}>Usar esta página</button><button class="btn ghost" disabled={busy} on:click={connect}>Volver a autorizar</button></div>
		{:else if data?.status === 'connected'}
			<p class="muted">Se comparten los artículos que se publiquen desde ahora; los anteriores no. Si un artículo queda en borrador, se comparte hasta que lo publiques.</p>
			<div class="acts"><button class="btn ghost" disabled={busy} on:click={connect}>Cambiar de página</button><button class="btn danger" disabled={busy} on:click={disconnect}>Desconectar</button></div>
		{:else if data?.status === 'broken'}
			<p class="error">{data.error || 'Facebook ya no deja publicar en tu página.'} Suele pasar si cambiaste tu contraseña de Facebook o quitaste el permiso.</p>
			<div class="acts"><button class="btn" disabled={busy} on:click={connect}>Volver a conectar</button><button class="btn danger" disabled={busy} on:click={disconnect}>Desconectar</button></div>
		{:else if data}
			<ul class="how">
				<li>Inicias sesión con Facebook y eliges la página de tu negocio.</li>
				<li>Cuando un artículo se publica, Faro lo comparte con un texto corto, la portada y el enlace.</li>
				<li>Solo publica en la página que elijas; nunca en tu perfil personal.</li>
			</ul>
			<button class="btn" disabled={busy} on:click={connect}>Conectar Facebook</button>
		{/if}
	</section>

	{#if data?.posts.length}
		<section class="panel">
			<h2 class="h-posts">Lo que se ha compartido</h2>
			<ul class="posts">
				{#each data.posts as p}
					<li>
						<div class="grow"><b>{p.title}</b><small>{when(p.at)}{#if p.error} · {p.error}{/if}</small></div>
						<span class="chip st-{p.status}">{statusName[p.status] || p.status}</span>
						{#if p.url}<a class="btn ghost small" href={p.url} target="_blank" rel="noopener noreferrer">Ver en Facebook</a>{/if}
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	<section class="panel soon">
		<div class="net-head">
			<span class="logo ig" aria-hidden="true"><svg viewBox="0 0 24 24"><rect x="3" y="3" width="18" height="18" rx="5" /><circle cx="12" cy="12" r="4" /><path d="M17.5 6.5h.01" /></svg></span>
			<div class="grow"><h2>Instagram, LinkedIn y Google Business</h2><small>Próximamente, sin costo extra</small></div>
		</div>
	</section>
</div>

<style>
	.redes { display: grid; gap: 22px; }
	.redes h1 { margin: 0; }
	.redes .sub { margin: 0 0 4px; }
	.net-head { display: flex; align-items: center; gap: 16px; }
	.net-head h2, .h-posts { margin: 0; font-size: 1.2rem; }
	.net-head small { color: var(--mut); }
	.net-head .bad { color: #a30b45; font-weight: 700; }
	.grow { flex: 1; min-width: 0; }
	.logo { display: grid; place-items: center; width: 48px; height: 48px; border-radius: 15px; flex: none; }
	.logo svg { width: 24px; height: 24px; }
	.fb { background: #1877f2; color: #fff; }
	.fb svg { fill: currentColor; }
	.ig { background: var(--accent-xl); color: var(--accent); }
	.ig svg { fill: none; stroke: currentColor; stroke-width: 2; stroke-linecap: round; }
	.net > p, .net > ul, .pages { margin: 20px 0 0; }
	.acts { display: flex; gap: 12px; flex-wrap: wrap; margin-top: 18px; }
	.net > .btn { margin-top: 18px; }
	.how { padding-left: 20px; color: #4a3a4e; line-height: 1.7; }
	.ok-chip { display: inline-flex; align-items: center; gap: 7px; background: #d6f5e3; color: #0f5a2e; }
	.ok-chip i { width: 7px; height: 7px; border-radius: 50%; background: #16a34a; }
	.pages { border: 0; padding: 0; display: grid; gap: 10px; }
	.pages legend { font-weight: 800; margin-bottom: 10px; }
	.page { display: flex; align-items: center; gap: 12px; padding: 14px 18px; border-radius: 18px; box-shadow: var(--out-s); font-weight: 700; cursor: pointer; }
	.page input { width: auto; box-shadow: none; accent-color: var(--accent); }
	.page.on { box-shadow: 0 0 0 2px var(--accent); background: var(--accent-xl); }
	.posts { list-style: none; margin: 14px 0 0; padding: 0; }
	.posts li { display: flex; align-items: center; gap: 14px; padding: 14px 0; flex-wrap: wrap; }
	.posts li + li { border-top: 1px solid var(--line); }
	.posts b { display: block; }
	.posts small { color: var(--mut); }
	.st-sent { background: #d6f5e3; color: #0f5a2e; }
	.st-failed, .st-unknown { background: #ffe0e8; color: #a30b45; }
	.st-cancelled { background: var(--well); color: var(--mut); }
	.soon { opacity: .75; }
</style>
