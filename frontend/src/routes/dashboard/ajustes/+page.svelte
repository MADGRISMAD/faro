<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { api } from '#lib/api';
	import { sites, siteId, research, user, refreshSite, loadSites, shortDate } from '#lib/app';

	let confirmText = '', removing = false;
	let error = '', notice = '', canonical = '', prefix = '/blog', loaded = '';
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
	$: left = $user ? ($user.budget_microusd - $user.spent_microusd - $user.reserved_microusd) / 1e6 : 0;
	async function removeSite() {
		if (!site) return;
		removing = true;
		error = notice = '';
		try {
			await api(`/sites/${site.id}?confirm=${encodeURIComponent(confirmText)}`, { method: 'DELETE' });
			confirmText = '';
			await loadSites();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			removing = false;
		}
	}
	let saved = '';
	async function perWeek(n: number) {
		error = '';
		saved = '';
		try {
			await api(`/sites/${$siteId}/autopilot`, { method: 'PATCH', body: JSON.stringify({ enabled: $research?.autopilot ?? true, posts_per_week: n }) });
			await refreshSite();
			const until = $research?.ramp_until;
			saved = until && n > 1 ? `Guardado. Hasta el ${shortDate(until)} sale 1 por semana (arranque); desde ese día, ${n} por semana. Tu calendario ya lo muestra.` : `Guardado: ${n} por semana. Tu calendario ya lo muestra.`;
		} catch (e) {
			error = (e as Error).message;
		}
	}
</script>

<div>
	<h1 class="text-2xl font-semibold tracking-tight">Ajustes</h1>
	<p class="text-sm text-muted-foreground">Integraciones y preferencias de {site?.domain}</p>
</div>
{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}
{#if notice}<Alert.Root role="status"><Alert.Description>{notice}</Alert.Description></Alert.Root>{/if}

<Card.Root>
	<Card.Header>
		<Card.Title>Conexión con tu sitio</Card.Title>
		<Card.Description>{$research?.connected ? 'Tu blog está conectado a tu dominio.' : 'Tu blog aún no está en tu dominio.'} <a class="underline" href="/dashboard/conectar">Ver cómo conectarlo →</a></Card.Description>
	</Card.Header>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title>Dirección del blog</Card.Title>
		<Card.Description>Si tu blog vive en otra ruta o subdominio, ajústalo aquí. Se usa en los enlaces, el sitemap y los datos para Google.</Card.Description>
	</Card.Header>
	<form onsubmit={(e) => { e.preventDefault(); (saveAddress)(); }}>
		<Card.Content class="flex flex-col gap-4">
			<div class="grid gap-2"><Label for="canonical">URL de tu sitio</Label><Input id="canonical" bind:value={canonical} required /></div>
			<div class="grid gap-2"><Label for="prefix">Ruta del blog</Label><Input id="prefix" bind:value={prefix} required /></div>
		</Card.Content>
		<Card.Footer class="mt-6"><Button type="submit" variant="secondary">Guardar dirección</Button></Card.Footer>
	</form>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title>Ritmo de publicación</Card.Title>
		<Card.Description>Un blog nuevo empieza con 1 por semana durante dos semanas; si ya tiene 5 o más artículos publicados, arranca directo. Nunca más de 5 por semana: un ritmo constante y moderado le sienta mejor a Google que publicar mucho de golpe.</Card.Description>
	</Card.Header>
	<Card.Content class="flex flex-col gap-4">
		<div class="flex flex-wrap gap-2">
			{#each [1, 2, 3, 5] as n}<Button size="sm" variant={$research?.posts_per_week === n ? 'default' : 'secondary'} onclick={() => perWeek(n)}>{n} por semana</Button>{/each}
		</div>
		{#if saved}<Alert.Root role="status"><Alert.Description>{saved}</Alert.Description></Alert.Root>{/if}
	</Card.Content>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title>Uso de IA</Card.Title>
		<Card.Description>Crédito disponible: <b class="text-foreground">US${left.toFixed(2)}</b> · consumido: US${($user?.spent_microusd / 1e6 || 0).toFixed(3)}. Cada artículo con su imagen cuesta alrededor de US$0.05.</Card.Description>
	</Card.Header>
</Card.Root>

<Card.Root>
	<Card.Header>
		<Card.Title class="text-destructive">Quitar este sitio</Card.Title>
		<Card.Description>Se borran sus artículos, imágenes y planes. No se puede deshacer. Lo que ya gastaste en IA se conserva en tu consumo. El blog que tengas en tu dominio dejará de recibir datos.</Card.Description>
	</Card.Header>
	<form onsubmit={(e) => { e.preventDefault(); (removeSite)(); }}>
		<Card.Content class="grid gap-2">
			<Label for="confirm">Escribe <b>{site?.domain}</b> para confirmar</Label>
			<Input id="confirm" bind:value={confirmText} autocomplete="off" autocapitalize="off" spellcheck={false} />
		</Card.Content>
		<Card.Footer class="mt-6"><Button type="submit" variant="destructive" disabled={removing || confirmText.trim().toLowerCase().replace(/^www\./, '') !== site?.domain}>{removing ? 'Quitando…' : 'Quitar este sitio'}</Button></Card.Footer>
	</form>
</Card.Root>
