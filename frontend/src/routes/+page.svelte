<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Item from '$lib/components/ui/item/index.js';
	import * as Accordion from '$lib/components/ui/accordion/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import { ArrowRight, Check, Search, PenLine, ImageIcon, ChartColumn, Share2, TrendingUp } from '@lucide/svelte';
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

<div class="mx-auto flex min-h-svh w-full max-w-6xl flex-col gap-16 px-4 pb-12 md:gap-24">
	<header class="flex flex-wrap items-center justify-between gap-3 py-4">
		<a href="/" class="flex flex-col leading-tight"><span class="text-lg font-semibold">Faro</span><span class="text-xs text-muted-foreground">by Hyperion</span></a>
		<nav class="hidden items-center gap-1 md:flex" aria-label="Secciones">
			<Button variant="ghost" size="sm" href="#como">Cómo funciona</Button>
			<Button variant="ghost" size="sm" href="#funciones">Funciones</Button>
			<Button variant="ghost" size="sm" href="#precios">Precios</Button>
			<Button variant="ghost" size="sm" href="#preguntas">Preguntas</Button>
		</nav>
		<div class="flex items-center gap-2">
			{#if signedIn}
				<Button size="sm" href="/dashboard">Ir a mi panel</Button>
			{:else}
				<Button variant="ghost" size="sm" href="/dashboard?modo=entrar">Iniciar sesión</Button>
				<Button size="sm" href="/dashboard">Crear cuenta</Button>
			{/if}
		</div>
	</header>

	<main class="flex flex-col gap-16 md:gap-24">
		<!-- Hero -->
		<section class="grid items-center gap-10 lg:grid-cols-2">
			<div class="flex flex-col items-start gap-5">
				<Badge variant="secondary">Un producto de Hyperion Software</Badge>
				<h1 class="text-4xl font-semibold tracking-tight md:text-6xl">Que te encuentren primero.</h1>
				<p class="text-lg text-muted-foreground">Cuando alguien busca lo que vendes, tu blog debería estar ahí. Pegas la dirección de tu sitio y la IA investiga tu negocio, elige los temas y escribe y publica artículos con imagen. Tú conservas el control y el diseño.</p>
				<div class="flex flex-wrap gap-2">
					{#if signedIn}
						<Button size="lg" href="/dashboard">Ir a mi panel<ArrowRight /></Button>
					{:else}
						<Button size="lg" href="/dashboard">Crear cuenta<ArrowRight /></Button>
						<Button size="lg" variant="outline" href="/dashboard?modo=entrar">Ya tengo cuenta · Iniciar sesión</Button>
					{/if}
				</div>
				<p class="text-sm text-muted-foreground">$200 MXN al mes · Un solo plan · Todo incluido</p>
			</div>

			<Card.Root>
				<Card.Header>
					<div class="relative"><Search class="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" /><Input readonly value="software a la medida en tijuana" aria-label="Búsqueda de ejemplo" class="pl-8" /></div>
				</Card.Header>
				<Card.Content class="flex flex-col gap-3">
					<Item.Root variant="outline">
						<Item.Content>
							<span><Badge>Escrito por Faro</Badge></span>
							<Item.Description>Hyperion Software · https://hyperionmkt.com › blog › software-a-la-medida-tijuana</Item.Description>
							<Item.Title class="text-base">Desarrollo de software a la medida en Tijuana: cómo cotizar tu proyecto</Item.Title>
							<Item.Description>hace 2 días — Qué incluye una cotización de software a la medida en Tijuana, cuánto tarda un proyecto y qué preguntar antes de contratar…</Item.Description>
						</Item.Content>
					</Item.Root>
					<Item.Root variant="muted">
						<Item.Content>
							<Item.Description>Directorio de empresas · https://directorio.example › tijuana › software</Item.Description>
							<Item.Title>Empresas de software en Tijuana, B.C. — Directorio 2026</Item.Title>
						</Item.Content>
					</Item.Root>
					<Item.Root variant="muted">
						<Item.Content>
							<Item.Description>Agencia Digital · https://agencia.example › servicios</Item.Description>
							<Item.Title>Desarrollo de software | Servicios para empresas</Item.Title>
						</Item.Content>
					</Item.Root>
				</Card.Content>
			</Card.Root>
		</section>

		<!-- Cómo funciona -->
		<section id="como" class="flex scroll-mt-8 flex-col gap-8">
			<div class="flex flex-col gap-2">
				<Badge variant="outline" class="w-fit">Cómo funciona</Badge>
				<h2 class="text-3xl font-semibold tracking-tight">De tu dirección web a artículos publicados</h2>
			</div>
			<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
				<Card.Root>
					<Card.Header><Badge variant="secondary" class="w-fit">Paso 01</Badge><Card.Title>Pega tu dirección</Card.Title><Card.Description>Escribe la URL de tu sitio. No instalas nada ni necesitas saber de SEO.</Card.Description></Card.Header>
					<Card.Content class="flex flex-col gap-2"><Input readonly value="hyperionmkt.com" aria-label="Dirección de ejemplo" /><Button size="sm" tabindex={-1} aria-hidden="true">Analizar sitio<ArrowRight /></Button></Card.Content>
				</Card.Root>
				<Card.Root>
					<Card.Header><Badge variant="secondary" class="w-fit">Paso 02</Badge><Card.Title>Investigamos tu negocio</Card.Title><Card.Description>La IA entiende qué vendes, a quién y cómo hablas, y lo guarda como tu Brand DNA.</Card.Description></Card.Header>
					<Card.Content class="flex flex-col gap-2 text-sm"><Badge variant="outline">Vende: software a la medida</Badge><Badge variant="outline">Para: pymes en Tijuana</Badge><Badge variant="outline">Tono: cercano y claro</Badge></Card.Content>
				</Card.Root>
				<Card.Root>
					<Card.Header><Badge variant="secondary" class="w-fit">Paso 03</Badge><Card.Title>Elegimos los temas</Card.Title><Card.Description>Palabras clave reales, cada una con su intención de búsqueda, acomodadas en tu calendario.</Card.Description></Card.Header>
					<Card.Content class="flex flex-col gap-2 text-sm">
						{#each [['punto de venta en la nube', 'Comparación'], ['software a la medida Tijuana', 'Local'], ['automatizar procesos', 'Informativo'], ['cotizar desarrollo web', 'Compra']] as [k, t]}
							<div class="flex items-center justify-between gap-2"><span>{k}</span><Badge variant="secondary">{t}</Badge></div>
						{/each}
					</Card.Content>
				</Card.Root>
				<Card.Root>
					<Card.Header><Badge variant="secondary" class="w-fit">Paso 04</Badge><Card.Title>Escribimos y publicamos</Card.Title><Card.Description>Artículos completos con imagen, publicados en tu blog. Tú revisas si quieres.</Card.Description></Card.Header>
					<Card.Content class="flex flex-col gap-1 text-sm"><span class="text-muted-foreground">hyperionmkt.com › blog</span><b>Software a la medida en Tijuana: cómo cotizar</b><span class="flex items-center gap-1 text-muted-foreground"><Check class="size-4" />Publicado hoy</span></Card.Content>
				</Card.Root>
			</div>

			<div class="grid gap-4 lg:grid-cols-2">
				<Card.Root>
					<Card.Header>
						<Badge variant="outline" class="w-fit">Tu blog crece solo</Badge>
						<Card.Title class="text-xl">24 artículos en 3 meses, sin escribir uno.</Card.Title>
						<Card.Description>Con el piloto automático a 2 artículos por semana, cada semana tu sitio suma nuevas páginas que pueden aparecer en Google.</Card.Description>
					</Card.Header>
					<Card.Content class="flex flex-col gap-4">
						<div class="flex flex-wrap gap-2"><Badge variant="secondary">2 por semana</Badge><Badge variant="secondary">~8 al mes</Badge><Badge variant="secondary">24 en 12 semanas</Badge></div>
						<div class="flex flex-col gap-1" role="img" aria-label="Calendario de ejemplo: dos publicaciones por semana durante 12 semanas">
							<div class="grid grid-cols-[1.5rem_repeat(12,1fr)] gap-1 text-xs text-muted-foreground"><span></span>{#each semanas as w}<span>{meses[w] ?? ''}</span>{/each}</div>
							{#each dias as d, r}
								<div class="grid grid-cols-[1.5rem_repeat(12,1fr)] gap-1"><span class="text-xs text-muted-foreground">{d}</span>{#each semanas as w}<i class="h-3 rounded-sm {r === 3 || r === 6 ? 'bg-primary' : 'bg-muted'}"></i>{/each}</div>
							{/each}
						</div>
					</Card.Content>
					<Card.Footer class="gap-4 text-xs text-muted-foreground"><span class="flex items-center gap-1"><i class="size-3 rounded-sm bg-primary"></i>Artículo publicado</span><span class="flex items-center gap-1"><i class="size-3 rounded-sm bg-muted"></i>Sin publicación</span></Card.Footer>
				</Card.Root>

				<Card.Root>
					<Card.Header>
						<Badge variant="outline" class="w-fit">Más visitas desde Google</Badge>
						<Card.Title class="text-xl">Cada artículo es una nueva puerta a tu sitio.</Card.Title>
						<Card.Description>Más páginas útiles significan más búsquedas en las que puedes aparecer. Publicar con constancia es lo que hace que el tráfico orgánico empiece a subir, y siga subiendo.</Card.Description>
					</Card.Header>
					<Card.Content class="flex flex-col gap-3">
						<div class="flex items-center justify-between text-sm"><b>Visitas desde Google</b><Badge variant="secondary"><TrendingUp />En aumento</Badge></div>
						<svg viewBox="0 0 600 232" preserveAspectRatio="none" class="h-40 w-full text-primary" aria-label="Gráfica de ejemplo: las visitas desde Google suben semana a semana" role="img">
							<path d="M0 22H600M0 80H600M0 139H600" stroke="currentColor" stroke-opacity=".15" stroke-width="1.3" stroke-dasharray="4 5" fill="none" />
							<path d="M0 198C60 193 90 190 125 187S190 180 224 173S290 158 330 148S400 126 444 103S490 84 520 68S570 38 600 18" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" vector-effect="non-scaling-stroke" />
						</svg>
						<div class="flex justify-between text-xs text-muted-foreground"><span>Semana 1</span><span>Semana 6</span><span>Semana 12</span></div>
					</Card.Content>
					<Card.Footer class="text-xs text-muted-foreground">Ejemplo ilustrativo. Los resultados dependen de tu giro y tu competencia.</Card.Footer>
				</Card.Root>
			</div>
		</section>

		<!-- Funciones -->
		<section id="funciones" class="flex scroll-mt-8 flex-col gap-8">
			<div class="flex flex-col gap-2">
				<Badge variant="outline" class="w-fit">Funciones</Badge>
				<h2 class="text-3xl font-semibold tracking-tight">Un equipo de contenido completo, en una sola pantalla</h2>
			</div>
			<div class="grid gap-4 lg:grid-cols-2">
				<Card.Root>
					<Card.Header><Card.Title>Palabras clave con intención</Card.Title><Card.Description>Cada artículo apunta a una búsqueda real: quien compara, quien busca cerca de ti, quien quiere aprender y quien ya va a comprar.</Card.Description></Card.Header>
					<Card.Content class="flex flex-col gap-2">{#each palabras as [k, t]}<Item.Root variant="outline" size="sm"><Item.Content><Item.Title>{k}</Item.Title></Item.Content><Item.Actions><Badge variant="secondary">{t}</Badge></Item.Actions></Item.Root>{/each}</Card.Content>
				</Card.Root>
				<Card.Root>
					<Card.Header><Card.Title>Calendario en automático</Card.Title><Card.Description>Dos artículos por semana, programados y publicados solos.</Card.Description></Card.Header>
					<Card.Content><div class="grid grid-cols-7 gap-1 text-center text-sm">{#each mes as d}<span class="grid h-9 place-items-center rounded-md {programados.includes(d) ? 'bg-primary font-medium text-primary-foreground' : 'bg-muted text-muted-foreground'}">{d}</span>{/each}</div></Card.Content>
				</Card.Root>
			</div>
			<div class="grid gap-4 md:grid-cols-3">
				<Card.Root><Card.Header><PenLine class="size-5 text-muted-foreground" /><Card.Title>Artículos con IA</Card.Title><Card.Description>Textos completos, estructurados para SEO y con la voz de tu marca.</Card.Description></Card.Header></Card.Root>
				<Card.Root><Card.Header><ImageIcon class="size-5 text-muted-foreground" /><Card.Title>Imagen incluida</Card.Title><Card.Description>Cada artículo se publica con su imagen, creada al escribirlo.</Card.Description></Card.Header></Card.Root>
				<Card.Root><Card.Header><ChartColumn class="size-5 text-muted-foreground" /><Card.Title>Analytics</Card.Title><Card.Description>Clics, impresiones y posiciones de cada palabra clave.</Card.Description></Card.Header></Card.Root>
			</div>
			<Card.Root>
				<Card.Header>
					<Share2 class="size-5 text-muted-foreground" />
					<Card.Title>Redes y Google Business</Card.Title>
					<Card.Description>Cada artículo compartido en tus redes y tu ficha de Google.</Card.Description>
					<Card.Action><Badge variant="secondary">Pronto</Badge></Card.Action>
				</Card.Header>
			</Card.Root>
		</section>

		<!-- Precios -->
		<section id="precios" class="flex scroll-mt-8 flex-col items-center gap-8">
			<div class="flex flex-col items-center gap-2 text-center">
				<Badge variant="outline" class="w-fit">Precios</Badge>
				<h2 class="text-3xl font-semibold tracking-tight">Un solo plan. Todo incluido.</h2>
			</div>
			<Card.Root class="w-full max-w-md">
				<Card.Header>
					<Card.Title>Faro <span class="text-sm font-normal text-muted-foreground">by Hyperion</span></Card.Title>
					<Card.Action><Badge variant="secondary">Mensual</Badge></Card.Action>
				</Card.Header>
				<Card.Content class="flex flex-col gap-4">
					<p><strong class="text-4xl font-semibold">$200</strong> <span class="text-muted-foreground">MXN / mes</span></p>
					<Separator />
					<ul class="flex flex-col gap-2 text-sm">
						{#each ['Artículos en piloto automático', 'Palabras clave y calendario', 'Imagen en cada artículo', 'Brand DNA y analytics'] as f}<li class="flex items-center gap-2"><Check class="size-4" />{f}</li>{/each}
						<li class="flex flex-wrap items-center gap-2"><Check class="size-4" />Redes sociales <Badge variant="secondary">Próximamente, sin costo extra</Badge></li>
					</ul>
				</Card.Content>
				<Card.Footer><Button class="w-full" href="/dashboard">{signedIn ? 'Ir a mi panel' : 'Crear cuenta'}</Button></Card.Footer>
			</Card.Root>
		</section>

		<!-- Preguntas -->
		<section id="preguntas" class="mx-auto flex w-full max-w-3xl scroll-mt-8 flex-col gap-6">
			<h2 class="text-3xl font-semibold tracking-tight">Preguntas frecuentes</h2>
			<Accordion.Root type="single">
				{#each preguntas as [q, a], i}
					<Accordion.Item value={`q${i}`}><Accordion.Trigger>{q}</Accordion.Trigger><Accordion.Content>{a}</Accordion.Content></Accordion.Item>
				{/each}
			</Accordion.Root>
		</section>

		<!-- Cierre -->
		<Card.Root>
			<Card.Header>
				<Card.Title class="text-2xl"><h2>Pega tu dirección. Nosotros escribimos.</h2></Card.Title>
				<Card.Description>Tu primer calendario de artículos queda listo en cuanto conectas tu sitio.</Card.Description>
			</Card.Header>
			<Card.Footer class="flex-wrap gap-2">
				{#if signedIn}
					<Button href="/dashboard">Ir a mi panel</Button>
				{:else}
					<Button href="/dashboard">Crear cuenta</Button><Button variant="outline" href="/dashboard?modo=entrar">Iniciar sesión</Button>
				{/if}
			</Card.Footer>
		</Card.Root>
	</main>

	<footer class="flex flex-wrap items-center justify-between gap-3 border-t pt-6 text-sm text-muted-foreground">
		<a href="/" class="flex flex-col leading-tight text-foreground"><span class="font-semibold">Faro</span><span class="text-xs text-muted-foreground">by Hyperion</span></a>
		<span>© 2026 <a class="underline" href="https://www.hyperionmkt.com/">Hyperion Software</a> · Tijuana, México</span>
		<nav aria-label="Legal" class="flex gap-4"><a class="underline" href="https://www.hyperionmkt.com/faro/privacidad">Privacidad</a><a class="underline" href="https://www.hyperionmkt.com/faro/terminos">Condiciones</a></nav>
	</footer>
</div>
