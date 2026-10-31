<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Item from '$lib/components/ui/item/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import { Checkbox } from '$lib/components/ui/checkbox/index.js';
	import { Progress } from '$lib/components/ui/progress/index.js';
	import { X } from '@lucide/svelte';
	import { api } from '#lib/api';
	import { research, siteId, refreshSite, imageStyles, intentName } from '#lib/app';

	// Asistente de la primera vez: la IA propone y el cliente ajusta. Nada se escribe hasta el último paso.
	const steps = ['Tu negocio', 'Tu público', 'Tu voz', 'Imágenes', 'Temas', 'Ritmo'];
	let step = 0, busy = false, error = '', ready = false;
	let business = '', themes: string[] = [], newTheme = '', audience = '', location = '', tone = '', avoid = '';
	let images = true, style = 'memphis', notes = '';
	let dropped: Record<string, boolean> = {};
	let perWeek = 2, autopilot = true, autoPublish = true;

	$: p = $research?.research;
	$: sensitive = !!p?.sensitive;
	// Los valores sugeridos se cargan una sola vez; después manda lo que el cliente va editando.
	$: if (p && !ready) {
		ready = true;
		business = p.business;
		themes = [...p.offerings];
		audience = p.audience;
		location = p.location;
		tone = p.tone;
		avoid = $research?.avoid_topics || '';
		images = $research?.images_enabled ?? true;
		style = $research?.image_style || 'memphis';
		notes = $research?.image_notes || '';
		perWeek = $research?.posts_per_week || 2;
		autoPublish = !sensitive;
	}
	$: kept = (p?.topics || []).filter((t) => !dropped[t.title]).length;

	async function save(body: Record<string, unknown>) {
		await api(`/sites/${$siteId}/brand`, { method: 'PATCH', body: JSON.stringify(body) });
	}
	async function next() {
		busy = true;
		error = '';
		try {
			if (step === 0) await save({ business, themes });
			if (step === 1) await save({ audience, location });
			if (step === 2) await save({ tone, avoid_topics: avoid });
			if (step === 3) await save({ images_enabled: images, image_style: style, image_notes: notes });
			if (step === 4) await save({ remove_topics: Object.keys(dropped).filter((k) => dropped[k]) });
			step += 1;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	async function finish() {
		busy = true;
		error = '';
		try {
			await save({ auto_publish: autoPublish });
			await api(`/sites/${$siteId}/autopilot`, { method: 'PATCH', body: JSON.stringify({ enabled: autopilot, posts_per_week: perWeek }) });
			await api(`/sites/${$siteId}/brand/confirm`, { method: 'POST' });
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
			busy = false;
		}
	}
	const addTheme = () => {
		if (newTheme.trim() && themes.length < 12) themes = [...themes, newTheme.trim()];
		newTheme = '';
	};
</script>

<main class="mx-auto flex min-h-svh w-full max-w-3xl flex-col justify-center gap-4 p-4 md:p-8">
	<div class="flex flex-col gap-2">
		<p class="text-sm text-muted-foreground">Brand DNA · paso {step + 1} de {steps.length}</p>
		<Progress value={((step + 1) / steps.length) * 100} max={100} aria-label="Avance del asistente" />
	</div>
	<Card.Root>
		<Card.Header>
			<Card.Title class="text-2xl"><h1>{steps[step]}</h1></Card.Title>
			<Card.Description>Esto es lo que la IA entendió de <b class="text-foreground">{$research?.domain}</b>. Está pensado para que solo corrijas lo que no te cuadre.</Card.Description>
			<Card.Action><Badge variant="secondary">Sugerido por la IA</Badge></Card.Action>
		</Card.Header>
		<Card.Content class="flex flex-col gap-4">
			{#if step === 0}
				<Label class="flex-col items-start">¿Qué hace tu negocio?<Textarea rows={5} bind:value={business} /></Label>
				<div class="flex flex-col gap-2">
					<b class="text-sm">Lo que ofreces</b>
					<div class="flex flex-wrap gap-2">{#each themes as t, i}<Badge variant="secondary" class="h-6 gap-1 pr-1">{t}<button type="button" class="rounded-full p-0.5 hover:bg-background" aria-label="Quitar {t}" onclick={() => (themes = themes.filter((_, j) => j !== i))}><X class="size-3" /></button></Badge>{/each}</div>
					<form class="flex gap-2" onsubmit={(e) => { e.preventDefault(); addTheme(); }}><Input bind:value={newTheme} placeholder="Agregar algo que ofreces" maxlength={100} aria-label="Agregar algo que ofreces" /><Button type="submit" variant="outline">Agregar</Button></form>
				</div>
			{:else if step === 1}
				<Label class="flex-col items-start">¿A quién le vendes?<Textarea rows={4} bind:value={audience} /></Label>
				<Label class="flex-col items-start">Ciudad o zona donde atiendes<Input bind:value={location} maxlength={160} /></Label>
			{:else if step === 2}
				<Label class="flex-col items-start">Voz y tono de tu marca<Input bind:value={tone} maxlength={160} /></Label>
				<Label class="flex-col items-start">Temas a evitar <span class="font-normal text-muted-foreground">(opcional: competidores, precios, servicios que no ofreces)</span><Textarea rows={3} bind:value={avoid} /></Label>
			{:else if step === 3}
				<div class="flex items-center justify-between gap-4">
					<div><b>Crear una imagen de portada para cada artículo</b><p class="text-sm text-muted-foreground">Ilustración sin texto. Cuesta unos US$0.05 por artículo con su imagen.</p></div>
					<Switch bind:checked={images} aria-label="Crear una imagen de portada para cada artículo" />
				</div>
				{#if images}
					<div class="grid grid-cols-2 gap-4 md:grid-cols-4">
						{#each imageStyles as s}
							<Item.Root variant="outline" class="items-stretch {style === s.id ? 'border-primary' : ''}">
								{#snippet child({ props })}
									<button {...props} type="button" class="{props.class} cursor-pointer flex-col text-left" aria-pressed={style === s.id} onclick={() => (style = s.id)}>
										<img class="aspect-video w-full rounded-md object-cover" src="/estilos/{s.id}.jpg" width="480" height="270" loading="lazy" alt="Muestra del estilo {s.name}" />
										<Item.Content><Item.Title>{s.name}</Item.Title><Item.Description>{s.hint}</Item.Description></Item.Content>
									</button>
								{/snippet}
							</Item.Root>
						{/each}
					</div>
					<Label class="flex-col items-start">Instrucciones de imagen <span class="font-normal text-muted-foreground">(opcional)</span><Input bind:value={notes} maxlength={300} placeholder="Por ejemplo: siempre incluir plantas, colores cálidos" /></Label>
				{/if}
			{:else if step === 4}
				<p class="text-sm text-muted-foreground">La IA escribirá sobre estos temas, uno por uno. Quita los que no quieras ({kept} seleccionados, mínimo 3).</p>
				{#each p?.topics || [] as t}
					<Label class="items-start justify-between gap-4 rounded-lg border p-3">
						<span class="flex items-start gap-3"><Checkbox class="mt-0.5" checked={!dropped[t.title]} onCheckedChange={(v) => (dropped = { ...dropped, [t.title]: !v })} aria-label={t.title} /><span class="flex flex-col gap-1 font-normal"><b class="font-medium">{t.title}</b><span class="text-sm text-muted-foreground">{t.why}</span></span></span>
						<Badge variant="secondary">{intentName[t.intent] || t.intent}</Badge>
					</Label>
				{/each}
			{:else}
				<div class="flex items-center justify-between gap-4">
					<div><b>Piloto automático</b><p class="text-sm text-muted-foreground">La IA escribe sola, a un ritmo moderado.</p></div>
					<Switch bind:checked={autopilot} aria-label="Piloto automático" />
				</div>
				{#if autopilot}
					<div class="flex flex-col gap-2">
						<b class="text-sm">Artículos por semana</b>
						<div class="flex gap-2">{#each [1, 2, 3] as n}<Button size="sm" variant={perWeek === n ? 'default' : 'secondary'} onclick={() => (perWeek = n)}>{n}</Button>{/each}</div>
						<p class="text-sm text-muted-foreground">Las primeras dos semanas sale 1 por semana: un ritmo constante le sienta mejor a Google que publicar mucho de golpe.</p>
					</div>
					<div class="flex items-center justify-between gap-4">
						<div><b>Publicar sin que yo revise</b><p class="text-sm text-muted-foreground">{sensitive ? 'Tu sector es sensible (' + (p?.sensitive_reason || 'salud, legal o finanzas') + '): Google pide revisión de un profesional, así que todo queda en borrador.' : 'Si lo apagas, cada artículo queda en borrador hasta que lo apruebes.'}</p></div>
						<Switch bind:checked={autoPublish} disabled={sensitive} aria-label="Publicar sin que yo revise" />
					</div>
				{/if}
			{/if}
			{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}
		</Card.Content>
		<Card.Footer class="justify-between">
			<Button variant="outline" onclick={() => (step -= 1)} disabled={step === 0 || busy}>Atrás</Button>
			{#if step < steps.length - 1}
				<Button onclick={next} disabled={busy || (step === 4 && kept < 3)}>{busy ? 'Guardando…' : 'Se ve bien, continuar'}</Button>
			{:else}
				<Button onclick={finish} disabled={busy}>{busy ? 'Iniciando…' : 'Empezar a escribir'}</Button>
			{/if}
		</Card.Footer>
	</Card.Root>
</main>
