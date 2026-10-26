<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '#lib/api';

	// Si ya hay una sesión abierta, los botones llevan directo al panel en lugar de pedir otra vez los datos.
	let signedIn = false;
	onMount(async () => {
		try {
			await api('/auth/me');
			signedIn = true;
		} catch {
			signedIn = false;
		}
	});

	const dias = ['L', 'M', 'M', 'J', 'V', 'S', 'D'];
	const semanas = Array.from({ length: 12 }, (_, i) => i);
	const meses: Record<number, string> = { 0: 'Nov', 5: 'Dic', 9: 'Ene' };
	const mes = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28];
	const programados = [1, 4, 8, 11, 15, 18, 22, 25];
	const palabras: [string, string][] = [
		['punto de venta en la nube', 'Comparación'],
		['desarrollo de software a la medida en Tijuana', 'Local'],
		['automatización de procesos para empresas', 'Informativo'],
		['cotizar desarrollo web', 'Compra']
	];
	const preguntas: [string, string][] = [
		['¿Necesito saber de SEO?', 'No. La IA investiga tu negocio, elige los temas y escribe pensando en lo que la gente busca en Google. Tú solo revisas si quieres.'],
		['¿Puedo revisar antes de que se publique?', 'Sí. Con el piloto automático apagado, o en sectores sensibles como salud, legal y finanzas, cada artículo queda en borrador hasta que lo apruebes.'],
		['¿Cambia el diseño de mi sitio?', 'No. Tu blog se conecta con el diseño de tu propio sitio por una API, o Faro lo aloja con la dirección de tu sitio. No tocamos el resto de tu página.'],
		['¿Las redes sociales cuestan aparte?', 'No. Cuando lleguen (próximamente), irán incluidas en el mismo plan, sin costo extra.']
	];
</script>

<svelte:head>
	<title>Faro by Hyperion · Que te encuentren primero</title>
	<meta name="description" content="Pega la dirección de tu sitio y Faro investiga tu negocio, elige los temas y escribe y publica artículos con imagen para que te encuentren en Google." />
</svelte:head>

<!-- Icono del faro: se repite en el logo y en el pedestal de la mascota -->
{#snippet faro()}
	<svg viewBox="0 0 64 64" aria-hidden="true">
		<path d="M23 20 7 14v13l16-3Z M41 20l16-6v13l-16-3Z" fill="#f7a8c4" />
		<rect x="26.5" y="15" width="11" height="9" rx="2.5" fill="#e8336d" />
		<path d="M27 24h10l3.2 29H23.8Z" fill="#c2185b" />
		<path d="M25.6 36h12.8l.6 5.6H25Z" fill="#f7a8c4" />
		<rect x="20" y="52" width="24" height="5" rx="2.5" fill="#9d1049" />
	</svg>
{/snippet}

<div class="faro">
	<div class="wrap">
		<header class="top">
			<a class="brand" href="/"><span class="logo-tile">{@render faro()}</span><span class="wordmark">Faro<small>by Hyperion</small></span></a>
			<nav class="links" aria-label="Secciones">
				<a href="#como">Cómo funciona</a><a href="#funciones">Funciones</a><a href="#precios">Precios</a><a href="#preguntas">Preguntas</a>
			</nav>
			<div class="acct">
				{#if signedIn}
					<a class="btn crimson sm" href="/dashboard">Ir a mi panel</a>
				{:else}
					<a class="login" href="/dashboard?modo=entrar">Iniciar sesión</a>
					<a class="btn crimson sm" href="/dashboard">Crear cuenta</a>
				{/if}
			</div>
		</header>

		<!-- Hero -->
		<section class="hero">
			<div class="hero-copy">
				<p class="eyebrow">Un producto de Hyperion Software</p>
				<h1>Que te <br />encuentren <br /><span>primero.</span></h1>
				<p class="lead">Cuando alguien busca lo que vendes, tu blog debería estar ahí. Pegas la dirección de tu sitio y la IA investiga tu negocio, elige los temas y escribe y publica artículos con imagen. Tú conservas el control y el diseño.</p>
				<div class="ctas">
					{#if signedIn}
						<a class="btn crimson lg" href="/dashboard">Ir a mi panel <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6" /></svg></a>
					{:else}
						<a class="btn crimson lg" href="/dashboard">Crear cuenta <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6" /></svg></a>
						<a class="btn soft lg" href="/dashboard?modo=entrar">Ya tengo cuenta · Iniciar sesión</a>
					{/if}
				</div>
				<p class="fine">$200 MXN al mes · Un solo plan · Todo incluido</p>
			</div>

			<div class="hero-art">
				<div class="serp">
					<div class="bar well"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="6.5" /><path d="m16 16 4.5 4.5" /></svg><b>software a la medida en tijuana</b><i class="caret"></i></div>
					<div class="tabs"><span class="on">Todo</span><span>Imágenes</span><span>Maps</span><span>Noticias</span><span>Más</span></div>
					<div class="res hit">
						<span class="badge">Escrito por Faro</span>
						<div class="site"><span class="fav dark">H</span><div><b>Hyperion Software</b><small>https://hyperionmkt.com › blog › software-a-la-medida-tijuana</small></div></div>
						<h3>Desarrollo de software a la medida en Tijuana: cómo cotizar tu proyecto</h3>
						<p><span>hace 2 días —</span> Qué incluye una cotización de <b>software a la medida</b> en <b>Tijuana</b>, cuánto tarda un proyecto y qué preguntar antes de contratar…</p>
					</div>
					<div class="res dim">
						<div class="site"><span class="fav">D</span><div><b>Directorio de empresas</b><small>https://directorio.example › tijuana › software</small></div></div>
						<h3 class="visited">Empresas de software en Tijuana, B.C. — Directorio 2026</h3>
						<p>Listado de agencias y desarrolladores en Tijuana. Compara servicios, ubicación y datos de contacto…</p>
					</div>
					<div class="res dimmer">
						<div class="site"><span class="fav">A</span><div><b>Agencia Digital</b><small>https://agencia.example › servicios</small></div></div>
						<h3 class="visited">Desarrollo de software | Servicios para empresas</h3>
					</div>
				</div>
				<div class="pedestal">{@render faro()}</div>
				<img class="hero-mascot" src="/mascota/animada.gif" width="260" height="260" alt="Mascota de Faro saludando" />
			</div>
		</section>

		<!-- Cómo funciona -->
		<section id="como" class="block">
			<p class="kicker">Cómo funciona</p>
			<h2 class="h2">De tu dirección web a artículos publicados</h2>
			<div class="steps">
				<article class="card step">
					<div class="well demo url-demo">
						<div class="field"><svg viewBox="0 0 24 24" class="lock" aria-hidden="true"><rect x="5" y="11" width="14" height="9" rx="2.5" /><path d="M8 11V8a4 4 0 0 1 8 0v3" /></svg><b>hyperionmkt.com</b><i class="caret"></i></div>
						<span class="mini-btn">Analizar sitio <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6" /></svg></span>
					</div>
					<p class="paso">Paso 01</p><h3>Pega tu dirección</h3><p class="desc">Escribe la URL de tu sitio. No instalas nada ni necesitas saber de SEO.</p>
					<span class="chev"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg></span>
				</article>
				<article class="card step">
					<div class="well demo dna-demo">
						<p class="mini-title">Brand DNA</p>
						<div class="pill-row"><b>Vende:</b> software a la medida</div>
						<div class="pill-row"><b>Para:</b> pymes en Tijuana</div>
						<div class="pill-row"><b>Tono:</b> cercano y claro</div>
					</div>
					<p class="paso">Paso 02</p><h3>Investigamos tu negocio</h3><p class="desc">La IA entiende qué vendes, a quién y cómo hablas, y lo guarda como tu Brand DNA.</p>
					<span class="chev"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg></span>
				</article>
				<article class="card step">
					<div class="well demo kw-demo">
						{#each [['punto de venta en la nube', 'Comparación'], ['software a la medida Tijuana', 'Local'], ['automatizar procesos', 'Informativo'], ['cotizar desarrollo web', 'Compra']] as [k, t]}
							<div class="kw-line"><span>{k}</span><em class="chip">{t}</em></div>
						{/each}
					</div>
					<p class="paso">Paso 03</p><h3>Elegimos los temas</h3><p class="desc">Palabras clave reales, cada una con su intención de búsqueda, acomodadas en tu calendario.</p>
				</article>
			</div>

			<div class="steps2">
				<article class="card step4">
					<div class="well demo pub-demo">
						<div class="pub-card">
							<span class="pub-thumb"><img src="/mascota/cruzado.png" width="760" height="428" alt="" /></span>
							<div><small>hyperionmkt.com › blog</small><b>Software a la medida en Tijuana: cómo cotizar</b><span class="ok"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5" /></svg> Publicado hoy</span></div>
						</div>
					</div>
					<p class="paso">Paso 04</p><h3>Escribimos y publicamos</h3><p class="desc">Artículos completos con imagen, publicados en tu blog. Tú revisas si quieres.</p>
				</article>
				<article class="card grows">
					<p class="kicker left">Tu blog crece solo</p>
					<h3 class="h3">24 artículos en 3 meses, sin escribir uno.</h3>
					<p class="lead2">Con el piloto automático a 2 artículos por semana, cada semana tu sitio suma nuevas páginas que pueden aparecer en Google.</p>
					<div class="pills"><span class="pill"><b>2</b> por semana</span><span class="pill"><b>~8</b> al mes</span><span class="pill"><b>24</b> en 12 semanas</span></div>
					<div class="well cal12">
						<div class="cal12-head"><b>Tu calendario, próximas 12 semanas</b><span class="pill"><i class="dot"></i><b>24 publicaciones</b></span></div>
						<div class="cal12-grid">
							<div class="months">{#each semanas as w}<span>{meses[w] ?? ''}</span>{/each}</div>
							{#each dias as d, r}
								<div class="cal12-row"><span class="dlabel">{d}</span>{#each semanas as w}<i class="sem" class:on={r === 3 || r === 6}></i>{/each}</div>
							{/each}
						</div>
						<p class="leyenda"><i class="sq on"></i> Artículo publicado <i class="sq"></i> Sin publicación</p>
					</div>
				</article>
			</div>

			<article class="card visitas">
				<div class="visitas-copy">
					<p class="kicker left">Más visitas desde Google</p>
					<h3 class="h3">Cada artículo es una nueva puerta a tu sitio.</h3>
					<p class="lead2">Más páginas útiles significan más búsquedas en las que puedes aparecer. Publicar con constancia es lo que hace que el tráfico orgánico empiece a subir, y siga subiendo.</p>
					<p class="key"><span class="keyicon"><i class="dot"></i></span> Artículo nuevo publicado</p>
					<p class="key"><span class="keyicon"><i class="dash"></i></span> Visitas orgánicas a tu sitio</p>
					<p class="note">Ejemplo ilustrativo. Los resultados dependen de tu giro y tu competencia.</p>
				</div>
				<div class="visitas-art">
					<img class="asombro" src="/mascota/asombro.png" width="400" height="430" alt="" />
					<span class="bubble">¡Mira cuántas visitas!</span>
					<div class="well chart">
						<div class="chart-head"><b>Visitas desde Google</b><span class="pill up"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m4 16 6-6 4 4 6-7M15 7h5v5" /></svg> En aumento</span></div>
						<div class="chart-box">
							<svg viewBox="0 0 600 232" preserveAspectRatio="none" aria-label="Gráfica de ejemplo: las visitas desde Google suben semana a semana" role="img">
								<defs><linearGradient id="area" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#e8336d" stop-opacity=".30" /><stop offset="1" stop-color="#e8336d" stop-opacity="0" /></linearGradient></defs>
								<path d="M0 22H600M0 80H600M0 139H600" stroke="#e5d3da" stroke-width="1.3" stroke-dasharray="4 5" fill="none" />
								<path d="M0 198C60 193 90 190 125 187S190 180 224 173S290 158 330 148S400 126 444 103S490 84 520 68S570 38 600 18L600 206L0 206Z" fill="url(#area)" stroke="none" />
								<path d="M0 198C60 193 90 190 125 187S190 180 224 173S290 158 330 148S400 126 444 103S490 84 520 68S570 38 600 18" fill="none" stroke="#c2185b" stroke-width="4" stroke-linecap="round" vector-effect="non-scaling-stroke" />
								<path d="M0 206H600" stroke="#dccad1" stroke-width="2" />
							</svg>
							<span class="endpoint"></span>
							<span class="tip">Más visitas <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 19V5M6 11l6-6 6 6" /></svg></span>
						</div>
						<div class="dots">{#each Array(20) as _}<i></i>{/each}</div>
						<div class="weeks"><span>Semana 1</span><span>Semana 6</span><span>Semana 12</span></div>
					</div>
				</div>
			</article>
		</section>

		<!-- Funciones -->
		<section id="funciones" class="block">
			<p class="kicker">Funciones</p>
			<h2 class="h2">Un equipo de contenido completo,<br />en una sola pantalla</h2>
			<div class="feat2">
				<article class="card">
					<h3 class="h3 s">Palabras clave con intención</h3>
					<p class="lead2">Cada artículo apunta a una búsqueda real: quien compara, quien busca cerca de ti, quien quiere aprender y quien ya va a comprar.</p>
					<div class="kwlist">{#each palabras as [k, t]}<div class="well kwrow"><b>{k}</b><em class="chip">{t}</em></div>{/each}</div>
				</article>
				<article class="card">
					<h3 class="h3 s">Calendario en automático</h3>
					<p class="lead2">Dos artículos por semana, programados y publicados solos.</p>
					<div class="well monthgrid">{#each mes as d}<span class:on={programados.includes(d)}>{d}</span>{/each}</div>
				</article>
			</div>
			<div class="feat3">
				<article class="card mini"><span class="ico"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 20h4L19 9l-4-4L4 16v4ZM13.5 6.5l4 4M14 20h6" /></svg></span><h3>Artículos con IA</h3><p>Textos completos, estructurados para SEO y con la voz de tu marca.</p></article>
				<article class="card mini"><span class="ico"><svg viewBox="0 0 24 24" aria-hidden="true"><rect x="4" y="5" width="16" height="14" rx="2.5" /><path d="m4.5 16 4.5-4.5 3.5 3.5 2.5-2.5 4.5 4.5M15.5 9.5h.01" /></svg></span><h3>Imagen incluida</h3><p>Cada artículo se publica con su imagen, creada al escribirlo.</p></article>
				<article class="card mini"><span class="ico"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 4v16h16M8 15l3-4 3 3 5-7" /></svg></span><h3>Analytics</h3><p>Clics, impresiones y posiciones de cada palabra clave.</p></article>
			</div>
			<article class="card wide">
				<span class="ico"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="6" cy="12" r="2.6" /><circle cx="17.5" cy="5.5" r="2.6" /><circle cx="17.5" cy="18.5" r="2.6" /><path d="m8.3 10.8 6.9-4M8.3 13.2l6.9 4" /></svg></span>
				<span class="chip soon">Pronto</span>
				<h3>Redes y Google Business</h3><p>Cada artículo compartido en tus redes y tu ficha de Google.</p>
			</article>
		</section>

		<!-- Precios -->
		<section id="precios" class="block pricing">
			<p class="kicker">Precios</p>
			<h2 class="h2">Un solo plan. Todo incluido.</h2>
			<div class="price-wrap">
				<img class="cruzado" src="/mascota/cruzado.png" width="760" height="428" alt="" />
				<article class="card price">
					<div class="price-head"><b>Faro <small class="by">by Hyperion</small></b><span class="pill m">Mensual</span></div>
					<p class="amount"><strong>$200</strong> <span>MXN / mes</span></p>
					<ul class="well checks">
						<li><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5" /></svg>Artículos en piloto automático</li>
						<li><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5" /></svg>Palabras clave y calendario</li>
						<li><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5" /></svg>Imagen en cada artículo</li>
						<li><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5" /></svg>Brand DNA y analytics</li>
						<li><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5" /></svg>Redes sociales <em class="chip soon">Próximamente, sin costo extra</em></li>
					</ul>
					<a class="btn crimson block-btn" href={signedIn ? '/dashboard' : '/dashboard'}>{signedIn ? 'Ir a mi panel' : 'Crear cuenta'}</a>
				</article>
			</div>
		</section>

		<!-- Preguntas -->
		<section id="preguntas" class="block faq">
			<h2 class="h2">Preguntas frecuentes</h2>
			<div class="faqs">
				{#each preguntas as [q, a]}
					<details class="card q"><summary><span>{q}</span><i class="plus"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg></i></summary><p>{a}</p></details>
				{/each}
			</div>
		</section>

		<!-- Cierre -->
		<section class="card final">
			<div class="final-art"><div class="final-circle"></div><img src="/mascota/saluda.png" width="440" height="440" alt="" /></div>
			<div>
				<h2 class="h2 left">Pega tu dirección. <span>Nosotros escribimos.</span></h2>
				<p class="lead2">Tu primer calendario de artículos queda listo en cuanto conectas tu sitio.</p>
				<div class="ctas">
					{#if signedIn}
						<a class="btn crimson" href="/dashboard">Ir a mi panel</a>
					{:else}
						<a class="btn crimson" href="/dashboard">Crear cuenta</a><a class="btn soft" href="/dashboard?modo=entrar">Iniciar sesión</a>
					{/if}
				</div>
			</div>
		</section>

		<footer class="foot">
			<a class="brand" href="/"><span class="logo-tile s">{@render faro()}</span><span class="wordmark s">Faro<small>by Hyperion</small></span></a>
			<span>© 2026 Hyperion Software · Tijuana, México</span>
			<nav aria-label="Legal"><a href="https://www.hyperionmkt.com/faro/privacidad">Privacidad</a><a href="https://www.hyperionmkt.com/faro/terminos">Condiciones</a></nav>
		</footer>
	</div>
</div>

<style>
	.faro{
		--bg:#fcfafb;--ink:#1f1424;--body:#5d4d61;--soft:#6b5a6e;--crimson:#c2185b;--pink:#e8336d;--tint:#ffe3ec;
		--sh:12px 16px 34px rgba(188,128,158,.27),-12px -12px 28px rgba(255,255,255,.96);
		--sh-s:6px 8px 18px rgba(188,128,158,.27),-6px -6px 14px rgba(255,255,255,.96);
		--in:inset 6px 7px 15px rgba(188,128,158,.24),inset -6px -6px 14px rgba(255,255,255,.96);
		--in-s:inset 3px 4px 9px rgba(188,128,158,.26),inset -3px -3px 8px rgba(255,255,255,.96);
		background:var(--bg);color:var(--ink);font-family:'Nunito',sans-serif;min-height:100vh;overflow-x:clip;
	}
	.faro :global(*){box-sizing:border-box}
	/* los estilos globales del panel (header, h1-h3, nav) no aplican aquí */
	.faro :global(h1),.faro :global(h2),.faro :global(h3){font-family:'Nunito',sans-serif;line-height:1.1}
	.faro :global(header.top){flex-wrap:nowrap;margin:0;padding:0 26px 0 16px;border-radius:40px}
	.faro :global(nav){gap:0}
	.faro :global(.links){gap:42px}
	.wrap{max-width:1240px;margin:0 auto;padding:24px 20px 40px}
	svg{fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round}
	a{color:inherit;text-decoration:none}
	.card{background:var(--bg);border-radius:40px;box-shadow:var(--sh)}
	.well{background:var(--bg);border-radius:30px;box-shadow:var(--in)}
	.chip{font-style:normal;font-weight:800;font-size:12px;color:var(--crimson);background:var(--tint);border-radius:99px;padding:4px 11px;white-space:nowrap}
	.pill{display:inline-flex;align-items:center;gap:7px;padding:9px 18px;border-radius:99px;box-shadow:var(--sh-s);font-size:15px;font-weight:700}
	.pill b{color:var(--crimson);font-weight:900}
	.dot{width:9px;height:9px;border-radius:50%;background:var(--pink);display:inline-block}
	.btn{display:inline-flex;align-items:center;justify-content:center;gap:10px;border-radius:99px;font-weight:800;transition:transform .15s}
	.btn:hover{transform:translateY(-1px)}
	.btn svg{width:18px;height:18px}
	.btn.crimson{background:var(--crimson);color:#fff;box-shadow:8px 10px 22px rgba(194,24,91,.34),-6px -6px 14px rgba(255,255,255,.95),inset 0 -3px 7px rgba(110,0,45,.28);padding:0 26px;height:46px;font-size:16px}
	.btn.soft{background:var(--bg);box-shadow:var(--sh-s);padding:0 28px;height:46px;font-size:16px}
	.btn.sm{height:44px;padding:0 24px;font-size:15.5px}
	.btn.lg{height:52px;padding:0 34px;font-size:17px}
	.btn:focus-visible,a:focus-visible,summary:focus-visible{outline:3px solid var(--crimson);outline-offset:3px}

	/* encabezado */
	.top{display:flex;align-items:center;justify-content:space-between;gap:20px;height:68px;padding:0 26px 0 16px;border-radius:40px;background:var(--bg);box-shadow:var(--sh)}
	.brand{display:flex;align-items:center;gap:14px}
	.logo-tile{display:grid;place-items:center;width:46px;height:46px;border-radius:15px;box-shadow:var(--sh-s);background:#fff}
	.logo-tile svg,.logo-tile.s svg{width:34px;height:34px;stroke:none}
	.logo-tile.s{width:40px;height:40px}
	.wordmark{font-family:Fraunces,serif;font-weight:600;font-size:28px;letter-spacing:-.01em;white-space:nowrap}.wordmark small,.by{font-family:'Nunito',sans-serif;font-size:.43em;font-weight:800;letter-spacing:.02em;color:var(--soft);margin-left:.55em}.by{font-size:12px;margin-left:2px}.wordmark.s{font-size:24px}
	.links{display:flex;gap:42px;font-weight:700;font-size:15.5px;white-space:nowrap}
	.acct{display:flex;align-items:center;gap:26px}.login{font-weight:700;font-size:15.5px;white-space:nowrap}.btn{white-space:nowrap}

	/* héroe */
	.hero{display:grid;grid-template-columns:minmax(0,1fr) 485px;gap:60px;padding:64px 6px 0;align-items:start}
	.eyebrow{margin:90px 0 0;color:var(--crimson);font-size:14px;font-weight:900;letter-spacing:.17em;text-transform:uppercase}
	h1{margin:22px 0 0;font-size:clamp(3.2rem,5.6vw,4.7rem);line-height:1.1;font-weight:900;letter-spacing:-.035em}
	h1 span{color:var(--crimson)}
	.lead{margin:30px 0 0;max-width:540px;font-size:19px;line-height:1.55;color:var(--body)}
	.ctas{display:flex;flex-wrap:wrap;gap:18px;margin-top:34px;align-items:center}
	.fine{margin:26px 0 0;font-size:15px;font-weight:700;color:var(--soft)}
	.hero-art{position:relative;margin-top:-36px}
	.serp{padding:26px;border-radius:40px;box-shadow:var(--sh);background:var(--bg);font-family:Arial,Helvetica,sans-serif}
	.serp .bar{display:flex;align-items:center;gap:12px;height:48px;padding:0 20px;border-radius:99px}
	.serp .bar svg{width:20px;height:20px;color:#5f5766}
	.serp .bar b{font-weight:700;font-size:16px;font-family:'Nunito',sans-serif;color:var(--ink)}
	.caret{width:2px;height:20px;background:var(--crimson);display:inline-block;margin-left:2px;animation:blink 1s steps(2) infinite}
	@keyframes blink{50%{opacity:0}}
	.tabs{display:flex;gap:26px;margin:22px 0 14px;padding:0 4px 0;border-bottom:1px solid #ecdde3;font-size:14px;color:#4d5156}
	.tabs span{padding:0 2px 10px}.tabs .on{color:#1a0dab;border-bottom:3px solid #1a0dab}
	.res{padding:18px 20px;border-radius:30px}
	.res.hit{position:relative;background:#fff;border:3px solid #f6a9c4;box-shadow:0 8px 20px rgba(194,24,91,.1)}
	.badge{position:absolute;right:16px;top:-14px;background:var(--crimson);color:#fff;border-radius:99px;font-family:'Nunito',sans-serif;font-weight:800;font-size:12.5px;padding:6px 14px}
	.site{display:flex;align-items:center;gap:12px}
	.site b{display:block;font-weight:400;font-size:14.5px;color:#202124}
	.site small{display:block;font-size:12.5px;color:#4d5156}
	.fav{display:grid;place-items:center;width:30px;height:30px;border-radius:50%;background:#eee3e8;font-weight:700;font-size:14px;color:#8f8290;flex:none}
	.fav.dark{background:#f3e9ed;color:#111}
	.res h3{margin:10px 0 6px;font-weight:400;font-size:19.5px;line-height:1.3;color:#1a0dab}
	.res h3.visited{color:#6c4fd1}
	.res p{margin:0;font-size:13.5px;line-height:1.55;color:#4d5156}.res p span{color:#70757a}.res p b{color:#202124}
	.res.dim{margin-top:14px;opacity:.78}.res.dimmer{opacity:.45;padding-bottom:6px}
	.pedestal{position:absolute;left:-139px;top:445px;width:184px;height:184px;border-radius:50%;box-shadow:var(--sh);background:var(--bg);display:grid;place-items:center;z-index:2}
	.pedestal svg{width:90px;height:90px;stroke:none;margin-top:44px}
	.pedestal::before{content:"";position:absolute;inset:14px;border-radius:50%;box-shadow:var(--in-s)}
	.hero-mascot{position:absolute;left:-196px;top:208px;width:250px;height:250px;z-index:3;filter:drop-shadow(0 14px 14px rgba(232,51,109,.2))}

	/* secciones */
	.block{padding-top:150px}
	.kicker{margin:0;text-align:center;color:var(--crimson);font-size:14px;font-weight:900;letter-spacing:.17em;text-transform:uppercase}
	.kicker.left{text-align:left}
	.h2{margin:14px auto 0;text-align:center;font-size:clamp(2.1rem,3.7vw,3.05rem);line-height:1.17;font-weight:900;letter-spacing:-.03em;max-width:760px}
	.h2.left{text-align:left;margin:0;max-width:none}
	.h3{margin:12px 0 0;font-size:clamp(1.7rem,2.9vw,2.4rem);line-height:1.15;font-weight:900;letter-spacing:-.025em}
	.h3.s{font-size:27px;margin:0}
	.lead2{margin:14px 0 0;font-size:17.5px;line-height:1.6;color:var(--body)}
	.steps{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:72px;margin-top:62px}
	.step{position:relative;padding:20px 20px 28px;border-radius:36px}
	.demo{height:160px;padding:18px;border-radius:26px;display:flex;flex-direction:column;justify-content:center;gap:10px;margin-bottom:22px}
	.url-demo .field{display:flex;align-items:center;gap:10px;background:#fff;border-radius:14px;height:38px;padding:0 14px;box-shadow:0 2px 6px rgba(188,128,158,.15)}
	.url-demo .field b{font-size:15px}.lock{width:16px;height:16px;color:#16a34a}
	.url-demo .caret{height:16px}
	.mini-btn{align-self:flex-end;display:inline-flex;align-items:center;gap:6px;background:var(--crimson);color:#fff;border-radius:11px;height:32px;padding:0 14px;font-weight:800;font-size:13.5px;box-shadow:0 6px 12px rgba(194,24,91,.25)}.mini-btn svg{width:13px;height:13px}
	.dna-demo{gap:6px;padding:14px 14px}
	.mini-title{margin:0;font-size:11.5px;font-weight:900;letter-spacing:.14em;text-transform:uppercase;color:#5d4d61}
	.pill-row{background:#fff;border-radius:11px;height:30px;display:flex;align-items:center;padding:0 14px;font-size:13.5px;box-shadow:0 2px 6px rgba(188,128,158,.13)}.pill-row b{color:var(--crimson);margin-right:5px;font-weight:800}
	.kw-demo{gap:9px;padding:14px 18px}.kw-line{display:flex;align-items:center;justify-content:space-between;gap:8px;font-size:13.5px}.kw-line .chip{font-size:11px;padding:3px 10px}
	.paso{margin:0;color:var(--crimson);font-size:14px;font-weight:900;letter-spacing:.15em;text-transform:uppercase}
	.step h3{margin:14px 0 8px;font-size:21px;font-weight:900}.desc{margin:0;font-size:15.5px;line-height:1.6;color:var(--body)}
	.chev{position:absolute;right:-56px;top:46%;width:40px;height:40px;border-radius:50%;background:var(--bg);box-shadow:var(--sh-s);display:grid;place-items:center;color:var(--crimson);z-index:2}
	.chev svg{width:16px;height:16px;stroke-width:3}
	.steps2{display:grid;grid-template-columns:326px minmax(0,1fr);gap:28px;margin-top:48px}
	.step4{padding:20px 20px 28px;border-radius:36px}
	.pub-demo{height:160px;padding:14px;justify-content:center}
	.pub-card{display:flex;gap:10px;background:#fff;border-radius:16px;padding:10px;box-shadow:0 2px 8px rgba(188,128,158,.15);align-items:center}
	.pub-thumb{flex:none;width:66px;height:86px;border-radius:11px;background:#fdd9e6;display:grid;place-items:center;overflow:hidden}.pub-thumb img{width:150%;max-width:none;height:auto;margin-top:6px}
	.pub-card small{display:block;font-size:11px;color:#6b6070}.pub-card b{display:block;font-size:13px;line-height:1.3;margin:2px 0 4px}
	.ok{display:inline-flex;align-items:center;gap:4px;font-size:12px;font-weight:800;color:#15803d}.ok svg{width:12px;height:12px;stroke-width:3}
	.grows{padding:34px 36px 30px}
	.pills{display:flex;flex-wrap:wrap;gap:14px;margin-top:26px}
	.cal12{margin-top:28px;padding:22px 24px 18px;border-radius:34px}
	.cal12-head{display:flex;align-items:center;justify-content:space-between;font-size:15px}.cal12-head .pill{padding:6px 14px;font-size:14px}
	.cal12-grid{margin-top:14px}
	.months,.cal12-row{display:grid;grid-template-columns:20px repeat(12,minmax(0,1fr));align-items:center;gap:0}
	.months{height:22px;margin-bottom:2px}.months span{font-size:11.5px;font-weight:900;color:var(--crimson);grid-column:auto;text-align:center}
	.months{grid-template-columns:20px repeat(12,minmax(0,1fr))}
	.months::before{content:"";}
	.cal12-row{height:27px}
	.dlabel{font-size:11.5px;font-weight:900;color:#5d4d61}
	.sem{justify-self:center;width:23px;height:23px;border-radius:8px;box-shadow:var(--in-s);display:block}
	.sem.on{background:var(--pink);box-shadow:3px 4px 9px rgba(232,51,109,.38),-3px -3px 7px rgba(255,255,255,.9)}
	.leyenda{display:flex;align-items:center;gap:8px;margin:12px 0 0;font-size:14px;font-weight:700;color:#4a3a4e}
	.sq{width:15px;height:15px;border-radius:5px;box-shadow:var(--in-s);display:inline-block}.sq.on{background:var(--pink);box-shadow:none}.sq+.sq{margin-left:10px}
	.visitas{margin-top:48px;display:grid;grid-template-columns:minmax(0,.9fr) minmax(0,1.1fr);gap:20px;padding:70px 48px 52px 48px;align-items:center;position:relative}
	.key{display:flex;align-items:center;gap:14px;margin:12px 0 0;font-size:16.5px;font-weight:700;color:#4a3a4e}
	.key:first-of-type{margin-top:26px}
	.keyicon{display:grid;place-items:center;width:40px;height:40px;border-radius:14px;box-shadow:var(--sh-s)}.dash{width:20px;height:5px;border-radius:3px;background:var(--crimson);display:block}.keyicon .dot{width:11px;height:11px}
	.note{margin:22px 0 0;font-size:13.5px;color:var(--soft)}
	.visitas-art{position:relative;padding-top:86px}
	.asombro{position:absolute;left:14px;top:-34px;width:104px;height:auto;z-index:3;filter:drop-shadow(0 10px 10px rgba(232,51,109,.22))}
	.bubble{position:absolute;left:138px;top:-14px;padding:14px 22px;border-radius:99px;box-shadow:var(--sh-s);font-weight:800;font-size:15px;z-index:3}
	.chart{padding:26px 28px 22px;border-radius:40px;position:relative}
	.chart-head{display:flex;align-items:center;justify-content:space-between;font-size:15px;color:#4a3a4e}
	.pill.up{color:#15803d;padding:8px 16px;font-size:14px}.pill.up svg{width:16px;height:16px}
	.chart-box{position:relative;margin:24px 8px 0}
	.chart-box>svg{width:100%;height:220px;display:block;overflow:visible}
	.endpoint{position:absolute;right:-9px;top:5px;width:22px;height:22px;border-radius:50%;background:#fff;border:5px solid var(--crimson);box-shadow:0 0 0 6px rgba(232,51,109,.18),0 0 16px rgba(232,51,109,.55)}
	.tip{position:absolute;right:48px;top:-14px;display:inline-flex;align-items:center;gap:6px;background:var(--ink);color:#fff;border-radius:99px;padding:9px 18px;font-weight:800;font-size:14.5px}.tip svg{width:13px;height:13px;stroke-width:3}
	.dots{display:flex;justify-content:space-between;margin:10px 8px 0}.dots i{width:9px;height:9px;border-radius:50%;background:var(--pink);box-shadow:0 2px 4px rgba(232,51,109,.35)}
	.weeks{display:flex;justify-content:space-between;margin:6px 8px 0;font-size:14.5px;font-weight:800;color:#4a3a4e}

	/* funciones */
	.feat2{display:grid;grid-template-columns:minmax(0,1.45fr) minmax(0,1fr);gap:29px;margin-top:62px}
	.feat2 .card{padding:40px 44px 40px;border-radius:44px}
	.kwlist{display:grid;gap:14px;margin-top:28px}.kwrow{display:flex;align-items:center;justify-content:space-between;height:52px;padding:0 20px;border-radius:99px;box-shadow:var(--in-s)}.kwrow b{font-size:16.5px;font-weight:800}
	.monthgrid{margin-top:26px;display:grid;grid-template-columns:repeat(7,1fr);gap:4px 0;padding:18px 18px;border-radius:34px}
	.monthgrid span{display:grid;place-items:center;height:48px;font-weight:800;font-size:15px;color:#5d4d61}
	.monthgrid span.on{background:var(--crimson);color:#fff;font-weight:900;border-radius:14px;width:62px;margin:0 auto;box-shadow:4px 6px 12px rgba(194,24,91,.3),-3px -3px 8px rgba(255,255,255,.9)}
	.feat3{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:28px;margin-top:48px}
	.mini{padding:34px 36px 38px;border-radius:44px}
	.ico{display:grid;place-items:center;width:52px;height:52px;border-radius:17px;box-shadow:var(--sh-s);color:var(--crimson)}.ico svg{width:24px;height:24px}
	.mini h3,.wide h3{margin:34px 0 14px;font-size:21px;font-weight:900}.mini p,.wide p{margin:0;font-size:16.5px;line-height:1.6;color:var(--body)}
	.wide{position:relative;margin-top:28px;padding:34px 36px 38px;border-radius:44px}
	.wide .chip.soon{position:absolute;right:36px;top:46px}
	.chip.soon{background:var(--tint);color:var(--crimson)}

	/* precios */
	.pricing{padding-top:120px}
	.price-wrap{position:relative;width:480px;max-width:100%;margin:200px auto 0}
	.cruzado{position:absolute;left:50%;top:-178px;width:264px;height:auto;transform:translateX(-50%);z-index:2;filter:drop-shadow(0 12px 12px rgba(232,51,109,.22))}
	.price{padding:60px 40px 36px;border-radius:44px}
	.price-head{display:flex;justify-content:space-between;align-items:center;font-size:18px;font-weight:900}.pill.m{padding:6px 18px;color:var(--crimson);font-weight:800;font-size:14.5px}
	.amount{margin:22px 0 0;line-height:1}.amount strong{font-size:86px;font-weight:900;color:var(--crimson);letter-spacing:-.04em}.amount span{font-size:18px;font-weight:700;color:var(--soft);margin-left:6px}
	.checks{list-style:none;margin:28px 0 0;padding:22px 24px;border-radius:30px;display:grid;gap:16px}
	.checks li{display:flex;align-items:center;gap:12px;font-size:17.5px;font-weight:700;color:#3c2d40}.checks svg{width:20px;height:20px;color:var(--crimson);stroke-width:3;flex:none}
	.checks li:last-child{flex-wrap:wrap}.checks .chip{margin-left:32px;font-size:12.5px}
	.block-btn{width:100%;height:56px;margin-top:30px;font-size:18px}

	/* preguntas */
	.faq{padding-top:110px}
	.faqs{display:grid;gap:19px;max-width:800px;margin:46px auto 0}
	.q{border-radius:34px}
	.q summary{list-style:none;cursor:pointer;display:flex;align-items:center;justify-content:space-between;gap:14px;min-height:72px;padding:0 24px 0 34px;font-size:18.5px;font-weight:800}
	.q summary::-webkit-details-marker{display:none}
	.plus{display:grid;place-items:center;width:34px;height:34px;border-radius:12px;box-shadow:var(--sh-s);color:var(--crimson);flex:none}.plus svg{width:15px;height:15px;stroke-width:3;transition:transform .2s}
	.q[open] .plus svg{transform:rotate(45deg)}
	.q p{margin:0;padding:0 34px 26px;font-size:16.5px;line-height:1.6;color:var(--body)}

	/* cierre y pie */
	.final{margin-top:130px;display:grid;grid-template-columns:230px minmax(0,1fr);gap:56px;align-items:center;padding:52px 64px 54px 66px;border-radius:50px}
	.final-art{position:relative;width:192px;height:192px}
	.final-circle{position:absolute;inset:0;border-radius:50%;box-shadow:var(--sh);background:var(--bg)}
	.final-circle::after{content:"";position:absolute;inset:12px;border-radius:50%;box-shadow:var(--in-s)}
	.final-art img{position:relative;width:128px;height:auto;left:28px;top:20px}
	.final .h2{font-size:clamp(2rem,3.7vw,3rem)}.final .h2 span{color:var(--crimson)}
	.final .lead2{margin-top:18px}.final .ctas{margin-top:22px;gap:16px}
	.foot{display:flex;align-items:center;justify-content:space-between;gap:20px;flex-wrap:wrap;margin-top:96px;padding:0 0 20px;font-size:15px;font-weight:700;color:#4a3a4e}
	.foot nav{display:flex;gap:22px}.foot nav a{text-decoration:underline}

	/* celular y tableta */
	@media(max-width:1100px){
		.links{display:none}.hero{grid-template-columns:minmax(0,1fr);gap:40px;padding-top:32px}.eyebrow{margin-top:30px}
		.hero-art{margin:80px 0 0;max-width:560px}.hero-mascot{left:-10px;top:-190px;width:200px;height:200px}.pedestal{display:none}
		.steps{grid-template-columns:minmax(0,1fr);gap:30px}.chev{display:none}.steps2,.feat2,.visitas,.feat3{grid-template-columns:minmax(0,1fr)}
		.final{grid-template-columns:minmax(0,1fr);padding:40px 30px}.final-art{margin:0 auto}
	}
	@media(max-width:640px){
		.wrap>:global(*){min-width:0}.hero-mascot{top:auto;bottom:calc(100% - 4px);left:auto;right:6px;width:130px;height:130px}.hero-art{margin-top:70px}.tip{right:auto;left:30%;top:-26px;padding:7px 12px;font-size:12.5px;white-space:nowrap}.note{margin-bottom:60px}.chart-head b{font-size:13.5px}.cal12-head{flex-wrap:wrap;gap:8px}.top .btn.sm{padding:0 16px}.pills{gap:8px}.pill{padding:8px 12px;font-size:13.5px}.eyebrow{font-size:12px}
		.wrap{padding:14px 12px 30px}.top{padding:0 12px;height:60px}.wordmark{font-size:22px}.login{display:none}.block{padding-top:96px}
		.cal12,.grows,.visitas{padding-left:16px;padding-right:16px}.cal12-row .sem{width:15px;height:15px;border-radius:5px}
		.chart-box>svg{height:170px}.visitas-art{padding-top:96px}.asombro{left:0}.bubble{left:112px;font-size:13.5px;padding:11px 16px}
		.price{padding:56px 22px 28px}.amount strong{font-size:68px}.checks li{font-size:15.5px;flex-wrap:wrap}.q summary{font-size:16.5px;padding-left:22px}
		.monthgrid span.on{width:40px}.feat2 .card,.mini,.wide{padding:28px 24px}
	}
	@media(prefers-reduced-motion:reduce){.caret{animation:none}.btn{transition:none}}
	:global(html){scroll-behavior:smooth}
</style>
