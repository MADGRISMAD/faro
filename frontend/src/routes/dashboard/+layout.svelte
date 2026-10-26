<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Wizard from '../../lib/Wizard.svelte';
	import { api } from '#lib/api';
	import { user, ready, sites, siteId, addingSite, research, boot, loadSites, selectSite, stopPolling } from '#lib/app';

	let mode: 'login' | 'register' = 'register';
	let email = '', password = '', url = '', error = '', busy = false;

	onMount(() => {
		here();
		if (new URLSearchParams(location.search).get('modo') === 'entrar') mode = 'login'; // viene del botón «Iniciar sesión» de la portada
		document.body.classList.add('in-app');
		boot();
	});
	onDestroy(() => {
		stopPolling();
		if (typeof document !== 'undefined') document.body.classList.remove('in-app');
	});

	async function authenticate() {
		busy = true;
		error = '';
		try {
			const r = await api(`/auth/${mode}`, { method: 'POST', body: JSON.stringify({ email, password }) });
			localStorage.setItem('faro_csrf', r.csrf);
			password = '';
			await boot();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	async function analyze() {
		busy = true;
		error = '';
		try {
			const r = await api('/sites/analyze', { method: 'POST', body: JSON.stringify({ url }) });
			url = '';
			siteId.set(r.site_id);
			addingSite.set(false);
			await loadSites();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	async function logout() {
		try {
			await api('/auth/logout', { method: 'POST' });
		} catch {}
		localStorage.removeItem('faro_csrf');
		user.set(null);
		sites.set([]);
		siteId.set('');
	}
	const nav = [
		['/dashboard', 'Inicio'],
		['/dashboard/brand', 'Brand DNA'],
		['/dashboard/calendario', 'Calendario'],
		['/dashboard/articulos', 'Artículos'],
		['/dashboard/conectar', 'Conectar mi sitio']
	];
	let path = '';
	const here = () => (path = location.pathname.replace(/\/$/, ''));
	$: active = $sites.find((s) => s.id === $siteId);
	// Un sitio que la IA acaba de analizar espera a que el cliente revise su Brand DNA antes de escribir nada.
	$: settingUp = !!$research && !$research.brand_confirmed;
	$: reading = settingUp && ($research?.status === 'queued' || $research?.status === 'running');
</script>

<svelte:head><title>FaroContenido</title><meta name="robots" content="noindex" /></svelte:head>
<svelte:window on:popstate={here} />

{#if !$ready}
	<main><p class="muted">Cargando…</p></main>
{:else if !$user}
	<main style="max-width:480px">
		<p class="eyebrow">FaroContenido</p>
		<h1 style="font-size:2.4rem">Tu blog, escrito solo.</h1>
		<div class="card">
			<div style="display:flex;gap:10px;margin-bottom:20px">
				<button class="btn" class:secondary={mode !== 'register'} on:click={() => (mode = 'register')}>Crear cuenta</button>
				<button class="btn" class:secondary={mode !== 'login'} on:click={() => (mode = 'login')}>Entrar</button>
			</div>
			<form class="stack" on:submit|preventDefault={authenticate}>
				<label>Correo<input type="email" bind:value={email} required autocomplete="email" /></label>
				<label>Contraseña<input type="password" bind:value={password} required minlength="12" autocomplete={mode === 'login' ? 'current-password' : 'new-password'} /></label>
				{#if error}<p class="error" role="alert">{error}</p>{/if}
				<button class="btn" disabled={busy}>{mode === 'register' ? 'Crear cuenta' : 'Entrar'}</button>
			</form>
		</div>
	</main>
{:else if !$sites.length || $addingSite}
	<main style="max-width:720px">
		<p class="eyebrow">Empieza aquí</p>
		<h1 style="font-size:2.6rem">Pega la dirección de tu sitio</h1>
		<p class="muted" style="font-size:1.05rem">La IA lee tu sitio, entiende tu negocio, elige los temas y empieza a escribir. No tienes que llenar nada más.</p>
		<form on:submit|preventDefault={analyze} style="display:flex;gap:10px;flex-wrap:wrap;margin-top:22px">
			<input bind:value={url} required maxlength="300" placeholder="tusitio.com" aria-label="Dirección de tu sitio" autocapitalize="off" autocorrect="off" spellcheck="false" style="flex:1;min-width:220px" />
			<button class="btn" disabled={busy}>{busy ? 'Enviando…' : 'Analizar mi sitio'}</button>
		</form>
		<p class="muted small" style="margin-top:10px">Sirve con o sin https, con www o sin él.</p>
		{#if error}<p class="error" role="alert">{error}</p>{/if}
		{#if $addingSite && $sites.length}<button class="btn secondary" style="margin-top:14px" on:click={() => addingSite.set(false)}>Cancelar</button>{/if}
	</main>
{:else if reading}
	<main style="max-width:640px">
		<p class="eyebrow">{active?.domain}</p>
		<h1 style="font-size:2.4rem">Leyendo tu sitio…</h1>
		<div class="panel" role="status"><span class="spin"></span>La IA está entendiendo tu negocio y eligiendo los temas. Tarda menos de un minuto; después podrás revisar todo antes de que escriba nada.</div>
	</main>
{:else if settingUp && $research?.status === 'failed'}
	<main style="max-width:640px">
		<h1 style="font-size:2.2rem">No pude leer tu sitio</h1>
		<p class="error" role="alert">{$research?.error}</p>
		<button class="btn" on:click={() => addingSite.set(true)}>Probar con otra dirección</button>
	</main>
{:else if settingUp && $research?.status === 'done'}
	<Wizard />
{:else}
	<div class="shell">
		<aside class="side">
			<a class="brand" href="/dashboard">FaroContenido<span>.</span></a>
			<div class="sitebox">
				<select aria-label="Sitio" value={$siteId} on:change={(e) => selectSite(e.currentTarget.value)}>
					{#each $sites as s}<option value={s.id}>{s.name}</option>{/each}
				</select>
				<small>{active?.domain}</small>
			</div>
			<nav>
				{#each nav as [href, label]}
					<a {href} class:on={path === href} on:click={() => (path = href)}>{label}{#if href === '/dashboard/conectar' && $research && !$research.connected}<span class="tag">Falta</span>{/if}</a>
				{/each}
				<span class="soon">Redes sociales <span class="tag">Pronto</span></span>
				<div class="group">Datos</div>
				<a href="/dashboard/analytics" class:on={path === '/dashboard/analytics'} on:click={() => (path = '/dashboard/analytics')}>Analytics</a>
				<a href="/dashboard/palabras-clave" class:on={path === '/dashboard/palabras-clave'} on:click={() => (path = '/dashboard/palabras-clave')}>Palabras clave</a>
			</nav>
			<div class="foot">
				<nav>
					<a href="/dashboard/ajustes" class:on={path === '/dashboard/ajustes'} on:click={() => (path = '/dashboard/ajustes')}>Ajustes</a>
					<a href="#nuevo" on:click|preventDefault={() => addingSite.set(true)}>+ Añadir otro sitio</a>
				</nav>
				<div class="me"><b>{($user.email || '?').slice(0, 2).toUpperCase()}</b><span>{$user.email}</span></div>
				<button class="btn secondary small" on:click={logout}>Cerrar sesión</button>
				<p class="legal"><a href="https://www.hyperionmkt.com/faro/privacidad" target="_blank" rel="noopener">Privacidad</a> · <a href="https://www.hyperionmkt.com/faro/terminos" target="_blank" rel="noopener">Condiciones</a></p>
			</div>
		</aside>
		<slot />
	</div>
{/if}
