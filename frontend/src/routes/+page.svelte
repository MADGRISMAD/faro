<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '#lib/api';

	// Si ya hay una sesión abierta, el botón lleva directo al panel en lugar de pedir otra vez los datos.
	let signedIn = false;
	onMount(async () => {
		try {
			await api('/auth/me');
			signedIn = true;
		} catch {
			signedIn = false;
		}
	});
	const steps = [
		['Pegas la dirección de tu sitio', 'Con o sin https, con o sin www. Sin formularios largos.'],
		['La IA entiende tu negocio', 'Deduce qué haces, a quién le vendes, tu tono y los temas que te conviene cubrir. Tú lo revisas en un asistente de pocos pasos.'],
		['Escribe y publica a un ritmo prudente', 'Un artículo con su imagen de portada, a un ritmo moderado que Google ve con buenos ojos. En sectores sensibles queda en borrador para que lo revises.'],
		['Lo ves en tu sitio, con tu diseño', 'Por una API de lectura, por tu propio dominio o en un blog alojado. Con Search Console conectado ves tus datos reales.']
	];
</script>

<svelte:head>
	<title>FaroContenido · Tu blog, escrito solo</title>
	<meta name="description" content="Pega la dirección de tu sitio y FaroContenido investiga tu negocio, elige los temas y escribe y publica artículos con imagen." />
</svelte:head>

<div class="landing">
	<header class="top">
		<a class="logo" href="/">FaroContenido<span>.</span></a>
		<nav aria-label="Cuenta">
			{#if signedIn}
				<a class="btn small" href="/dashboard">Ir a mi panel</a>
			{:else}
				<a class="btn ghost small" href="/dashboard?modo=entrar">Iniciar sesión</a>
				<a class="btn small" href="/dashboard">Crear cuenta</a>
			{/if}
		</nav>
	</header>

	<main>
		<p class="eyebrow">Un producto de Hyperion Software</p>
		<h1>Tu blog, escrito solo.</h1>
		<p class="lead">Pegas la dirección de tu sitio y la inteligencia artificial investiga tu negocio, elige los temas y escribe y publica artículos con imagen. Tú conservas el control y el diseño.</p>
		<div class="cta">
			{#if signedIn}
				<a class="btn" href="/dashboard">Ir a mi panel</a>
			{:else}
				<a class="btn" href="/dashboard">Crear cuenta</a>
				<a class="btn ghost" href="/dashboard?modo=entrar">Ya tengo cuenta · Iniciar sesión</a>
			{/if}
		</div>

		<ol class="steps">
			{#each steps as [title, text], i}
				<li><span class="n">0{i + 1}</span><h2>{title}</h2><p class="muted">{text}</p></li>
			{/each}
		</ol>
	</main>

	<footer class="foot">
		<span>© 2026 Hyperion Software · Tijuana, México</span>
		<nav aria-label="Legal">
			<a href="https://www.hyperionmkt.com/faro/privacidad">Política de privacidad</a>
			<a href="https://www.hyperionmkt.com/faro/terminos">Condiciones del servicio</a>
			<a href="mailto:info@hyperionmkt.com">Contacto</a>
		</nav>
	</footer>
</div>

<style>
	.landing{min-height:100vh;display:flex;flex-direction:column;padding:18px max(18px,calc((100vw - 1100px)/2)) 0}
	.top{display:flex;align-items:center;justify-content:space-between;gap:16px;margin:0;padding:12px 22px;border-radius:26px;background:var(--surface);box-shadow:var(--out)}
	.logo{font-family:Fraunces,serif;font-size:1.35rem;font-weight:600;text-decoration:none}.logo span{color:var(--accent)}
	.top nav{display:flex;gap:10px;align-items:center}
	main{flex:1;max-width:none;margin:0;padding:64px 6px 40px}
	h1{font-size:clamp(2.8rem,8vw,5.6rem);line-height:.98;margin:.35rem 0 0;letter-spacing:-.03em}
	.lead{max-width:40rem;margin:1.4rem 0 0;font-size:1.2rem;line-height:1.6;color:var(--mut)}
	.cta{display:flex;gap:14px;flex-wrap:wrap;margin-top:2rem}
	.steps{list-style:none;margin:72px 0 0;padding:0;display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:22px}
	.steps li{padding:24px;border-radius:28px;background:var(--surface);box-shadow:var(--out)}
	.n{display:grid;place-items:center;width:46px;height:46px;border-radius:15px;background:var(--well);box-shadow:var(--in-s);font-family:Fraunces,serif;color:var(--accent-d)}
	.steps h2{font-size:1.25rem;margin:16px 0 6px}.steps p{margin:0;line-height:1.55}
	.foot{display:flex;flex-wrap:wrap;gap:14px;justify-content:space-between;padding:26px 6px 30px;font-size:.88rem;color:var(--mut)}
	.foot nav{display:flex;gap:20px;flex-wrap:wrap}.foot a{text-decoration:underline}
</style>
