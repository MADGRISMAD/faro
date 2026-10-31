<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Item from '$lib/components/ui/item/index.js';
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Textarea } from '$lib/components/ui/textarea/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Switch } from '$lib/components/ui/switch/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { Globe, MapPin, MessageSquare, Users, Tag, Ban, ImageIcon, RefreshCw, Check, X } from '@lucide/svelte';
	import { api } from '#lib/api';
	import { research, siteId, sites, refreshSite, imageStyles } from '#lib/app';

	let askReread = false;
	let editing = '', draft = '', error = '', saved = '', newTheme = '', notes = '', notesSite = '';
	$: p = $research?.research;
	$: domain = $sites.find((s) => s.id === $siteId)?.domain || '';
	$: reading = $research?.status === 'queued' || $research?.status === 'running';
	$: style = imageStyles.find((s) => s.id === $research?.image_style) || imageStyles[0];
	// las instrucciones de imagen se editan en línea; se recargan al cambiar de sitio
	$: if ($siteId !== notesSite && $research) {
		notesSite = $siteId;
		notes = $research.image_notes || '';
	}

	async function patch(body: Record<string, unknown>) {
		error = '';
		try {
			await api(`/sites/${$siteId}/brand`, { method: 'PATCH', body: JSON.stringify(body) });
			saved = 'Guardado';
			setTimeout(() => (saved = ''), 1500);
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		}
	}
	function edit(key: string, value: string) {
		editing = key;
		draft = value;
	}
	async function commit() {
		await patch({ [editing]: draft });
		editing = '';
	}
	const addTheme = async () => {
		if (!newTheme.trim() || !p) return;
		await patch({ themes: [...p.offerings, newTheme.trim()] });
		newTheme = '';
	};
	async function reread() {
		error = '';
		try {
			await api(`/sites/${$siteId}/research`, { method: 'POST' });
			await refreshSite();
		} catch (e) {
			error = (e as Error).message;
		}
	}

	// «Profesional, innovador y accesible» → chips; un «en México» al final se muestra aparte
	const cap = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);
	function split(text: string) {
		const parts = text.replace(/\.$/, '').split(/,\s*|\s+y\s+|;\s*/).map((s) => s.trim()).filter(Boolean);
		let where = '';
		const m = parts.length ? parts[parts.length - 1].match(/^(.*\S)\s+en\s+(.+)$/) : null;
		if (m) {
			parts[parts.length - 1] = m[1];
			where = 'En ' + m[2];
		}
		return { items: parts.map(cap), where };
	}
	$: tone = split(p?.tone || '');
	$: audience = split(p?.audience || '');
</script>

{#snippet editBtn(key: string, value: string)}
	{#if editing === key}
		<span class="flex gap-2"><Button variant="ghost" size="sm" onclick={() => (editing = '')}>Cancelar</Button><Button size="sm" onclick={commit}>Guardar</Button></span>
	{:else}
		<Button variant="ghost" size="sm" onclick={() => edit(key, value)}>{value ? 'Editar' : 'Agregar'}</Button>
	{/if}
{/snippet}

<div class="flex flex-wrap items-end justify-between gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Brand DNA</h1>
		<p class="text-sm text-muted-foreground">Así entiende la IA tu marca. Todo lo que escribe sale de aquí.</p>
	</div>
	<div class="flex flex-wrap items-center gap-2">
		{#if saved}<Badge>{saved}</Badge>{/if}
		{#if p}<Badge variant="outline">{#if reading}<Spinner class="size-3" />{/if}{reading ? `Leyendo ${domain}…` : `Aprendido de ${domain}`}</Badge>{/if}
		<Button variant="outline" size="sm" disabled={reading} onclick={() => (askReread = true)}><RefreshCw />Volver a leer mi sitio</Button>
	</div>
</div>
{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}

<AlertDialog.Root bind:open={askReread}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>¿Volver a leer tu sitio?</AlertDialog.Title>
			<AlertDialog.Description>La IA volverá a leer {domain} y reemplazará lo que ves aquí (tipo de negocio, tono, público y temas).</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancelar</AlertDialog.Cancel>
			<AlertDialog.Action onclick={reread}>Continuar</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>

{#if !p && $research?.status === 'none'}
	<Alert.Root><Alert.Description>Este sitio se dio de alta a mano, así que la IA no lo ha leído todavía. Usa «Volver a leer mi sitio» y la IA llenará aquí todo por ti.</Alert.Description></Alert.Root>
{:else if !p}
	<Alert.Root role="status"><Spinner /><Alert.Description>Todavía estoy analizando tu sitio…</Alert.Description></Alert.Root>
{:else}
	<!-- Tipo de negocio, idioma y zona -->
	<div class="grid gap-4 lg:grid-cols-3">
		<Card.Root class="lg:col-span-2">
			<Card.Header>
				<Card.Description>Tipo de negocio</Card.Description>
				<Card.Action>{@render editBtn('business', p.business)}</Card.Action>
			</Card.Header>
			<Card.Content>
				{#if editing === 'business'}<Textarea bind:value={draft} rows={4} aria-label="Tipo de negocio" />{:else}<p class="text-xl font-semibold leading-snug">{p.business}</p>{/if}
			</Card.Content>
		</Card.Root>
		<Card.Root>
			<Card.Content class="flex flex-col gap-3">
				<Item.Root variant="outline">
					<Item.Media variant="icon"><Globe /></Item.Media>
					<Item.Content><Item.Description>Idioma del contenido</Item.Description><Item.Title>Español (México)</Item.Title></Item.Content>
				</Item.Root>
				<Item.Root variant="outline">
					<Item.Media variant="icon"><MapPin /></Item.Media>
					<Item.Content>
						<Item.Description>Zona de servicio · opcional</Item.Description>
						{#if editing === 'location'}<Input bind:value={draft} maxlength={120} placeholder="Ciudad o zona que atiendes" aria-label="Zona de servicio" />{:else}<Item.Title>{p.location || 'Sin zona'}</Item.Title>{/if}
					</Item.Content>
					<Item.Actions>{@render editBtn('location', p.location || '')}</Item.Actions>
				</Item.Root>
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Voz y público -->
	<div class="grid gap-4 md:grid-cols-2">
		<Card.Root>
			<Card.Header>
				<Card.Title class="flex items-center gap-2"><MessageSquare class="size-4" />Voz y tono</Card.Title>
				<Card.Description>Cómo suenan tus artículos</Card.Description>
				<Card.Action>{@render editBtn('tone', p.tone)}</Card.Action>
			</Card.Header>
			<Card.Content>
				{#if editing === 'tone'}<Textarea bind:value={draft} rows={3} aria-label="Voz y tono" />
				{:else}<div class="flex flex-wrap gap-2">{#each tone.items as t}<Badge variant="secondary">{t}</Badge>{/each}</div>{/if}
			</Card.Content>
		</Card.Root>
		<Card.Root>
			<Card.Header>
				<Card.Title class="flex items-center gap-2"><Users class="size-4" />Público objetivo</Card.Title>
				<Card.Description>Para quién escribimos</Card.Description>
				<Card.Action>{@render editBtn('audience', p.audience)}</Card.Action>
			</Card.Header>
			<Card.Content>
				{#if editing === 'audience'}<Textarea bind:value={draft} rows={3} aria-label="Público objetivo" />
				{:else}<div class="flex flex-wrap gap-2">{#each audience.items as a}<Badge variant="outline">{a}</Badge>{/each}</div>{#if audience.where}<p class="mt-3 text-sm text-muted-foreground">{audience.where}</p>{/if}{/if}
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Temas -->
	<div class="grid gap-4 lg:grid-cols-3">
		<Card.Root class="lg:col-span-2">
			<Card.Header>
				<Card.Title class="flex items-center gap-2"><Tag class="size-4" />Temas clave <span class="text-muted-foreground">· {p.offerings.length}</span></Card.Title>
				<Card.Description>Tus artículos giran alrededor de estos temas</Card.Description>
			</Card.Header>
			<Card.Content class="flex flex-col gap-4">
				<div class="flex flex-wrap gap-2">
					{#each p.offerings as t, i}
						<Badge variant="secondary" class="h-6 gap-1 pr-1">{t}<button type="button" class="rounded-full p-0.5 hover:bg-background" aria-label="Quitar {t}" onclick={() => patch({ themes: p.offerings.filter((_, j) => j !== i) })}><X class="size-3" /></button></Badge>
					{/each}
				</div>
				<form class="flex gap-2" onsubmit={(e) => { e.preventDefault(); (addTheme)(); }}>
					<Input bind:value={newTheme} placeholder="Agregar tema clave…" maxlength={100} aria-label="Nuevo tema clave" />
					<Button type="submit" disabled={!newTheme.trim()}>Agregar</Button>
				</form>
			</Card.Content>
		</Card.Root>
		<Card.Root>
			<Card.Header>
				<Card.Title class="flex items-center gap-2"><Ban class="size-4" />Temas a evitar</Card.Title>
				<Card.Description>Opcional</Card.Description>
				{#if $research?.avoid_topics || editing === 'avoid_topics'}<Card.Action>{@render editBtn('avoid_topics', $research?.avoid_topics || '')}</Card.Action>{/if}
			</Card.Header>
			<Card.Content class="flex flex-col items-start gap-2">
				{#if editing === 'avoid_topics'}<Textarea bind:value={draft} rows={4} aria-label="Temas a evitar" placeholder="Por ejemplo: nombres de competidores, precios, servicios que no ofreces" />
				{:else if $research?.avoid_topics}<p class="whitespace-pre-wrap font-medium">{$research.avoid_topics}</p><p class="text-sm text-muted-foreground">La IA nunca mencionará esto.</p>
				{:else}
					<p class="text-sm">La IA nunca mencionará lo que pongas aquí.</p>
					<p class="text-sm text-muted-foreground">Por ejemplo: nombres de competidores, precios, servicios que no ofreces.</p>
					<Button variant="outline" size="sm" onclick={() => edit('avoid_topics', '')}>+ Agregar tema a evitar</Button>
				{/if}
			</Card.Content>
		</Card.Root>
	</div>

	<!-- Imágenes -->
	<Card.Root>
		<Card.Header>
			<Card.Title class="flex items-center gap-2"><ImageIcon class="size-4" />Imágenes de tus artículos</Card.Title>
			<Card.Description>Una portada por artículo, sin texto. Cuesta unos US$0.045 por imagen.</Card.Description>
			<Card.Action>
				<label class="flex cursor-pointer items-center gap-2 text-sm">{$research?.images_enabled ? 'Activado' : 'Apagado'}<Switch checked={$research?.images_enabled} onCheckedChange={(v) => patch({ images_enabled: v })} aria-label="Imágenes de tus artículos" /></label>
			</Card.Action>
		</Card.Header>
		{#if $research?.images_enabled}
			<Card.Content class="flex flex-col gap-6">
				<div class="flex flex-wrap items-center justify-between gap-2 text-sm"><span class="text-muted-foreground">Estilo de imagen</span><span>Elegido: <b>{style.name}</b></span></div>
				<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
					{#each imageStyles as s}
						<Item.Root variant="outline" class="items-stretch {$research?.image_style === s.id ? 'border-primary' : ''}">
							{#snippet child({ props })}
								<button {...props} type="button" class="{props.class} cursor-pointer flex-col text-left" aria-pressed={$research?.image_style === s.id} onclick={() => patch({ image_style: s.id })}>
									<span class="relative block w-full">
										<img class="aspect-video w-full rounded-md object-cover" src="/estilos/{s.id}.jpg" width="480" height="270" loading="lazy" alt="Muestra del estilo {s.name}" />
										{#if $research?.image_style === s.id}<span class="absolute top-2 right-2 grid size-6 place-items-center rounded-full bg-primary text-primary-foreground"><Check class="size-3.5" /></span>{/if}
									</span>
									<Item.Content><Item.Title>{s.name}</Item.Title><Item.Description>{s.hint}</Item.Description></Item.Content>
								</button>
							{/snippet}
						</Item.Root>
					{/each}
				</div>
				<form class="grid gap-2" onsubmit={(e) => { e.preventDefault(); (() => patch({ image_notes: notes }))(); }}>
					<Label for="notes">Instrucciones de imagen <span class="text-muted-foreground">· opcional</span></Label>
					<div class="flex gap-2"><Input id="notes" bind:value={notes} maxlength={300} placeholder="Por ejemplo: siempre incluir plantas, colores cálidos" /><Button type="submit" variant="outline" disabled={notes === ($research?.image_notes || '')}>Guardar</Button></div>
				</form>
			</Card.Content>
		{/if}
	</Card.Root>
{/if}
