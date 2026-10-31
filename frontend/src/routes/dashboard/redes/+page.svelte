<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Item from '$lib/components/ui/item/index.js';
	import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
	import { RadioGroup, RadioGroupItem } from '$lib/components/ui/radio-group/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Share2 } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { api } from '#lib/api';
	import { siteId, sites } from '#lib/app';

	type Post = { title: string; status: string; url: string; error: string; at: string };
	type Data = { configured: boolean; status: 'none' | 'choosing' | 'connected' | 'broken'; page_name?: string; error?: string; choices: { id: string; name: string }[]; posts: Post[]; unshared: { id: string; title: string; published_at: string }[] };
	let askDisconnect = false;
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
	const share = (ids: string[]) => run(async () => {
		const r = await api(`/sites/${$siteId}/facebook/share`, { method: 'POST', body: JSON.stringify({ article_ids: ids }) });
		notice = r.queued > 1 ? `Listo: ${r.queued} artículos en cola. El primero sale en un minuto y los demás uno cada 2 horas, para no llenar tu página de golpe.` : 'Listo: sale en tu página en un minuto.';
		await load();
	});
	const disconnect = () => run(async () => {
		await api(`/sites/${$siteId}/facebook`, { method: 'DELETE' });
		notice = '';
		await load();
	});
	const when = (s: string) => new Date(s).toLocaleDateString('es-MX', { day: 'numeric', month: 'short', hour: 'numeric', minute: '2-digit' });
</script>

<div>
	<h1 class="text-2xl font-semibold tracking-tight">Redes sociales</h1>
	<p class="text-sm text-muted-foreground">Cada artículo que Faro publique en {domain || 'tu sitio'} se comparte solo en tus redes, con su portada y el enlace al artículo.</p>
</div>
{#if notice}<Alert.Root role="status"><Alert.Description>{notice}</Alert.Description></Alert.Root>{/if}
{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}

<AlertDialog.Root bind:open={askDisconnect}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>¿Desconectar Facebook?</AlertDialog.Title>
			<AlertDialog.Description>Faro dejará de compartir tus artículos en Facebook. Lo que ya se publicó en tu página no se borra.</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancelar</AlertDialog.Cancel>
			<AlertDialog.Action onclick={disconnect}>Desconectar</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>

<Card.Root>
	<Card.Header>
		<Card.Title class="flex items-center gap-2"><Share2 class="size-5" />Facebook</Card.Title>
		<Card.Description>
			{#if !data}Cargando…
			{:else if data.status === 'connected'}Conectado a <b class="text-foreground">{data.page_name}</b>
			{:else if data.status === 'broken'}<span class="text-destructive">La conexión dejó de funcionar</span>
			{:else}Comparte cada artículo en la página de tu negocio{/if}
		</Card.Description>
		{#if data?.status === 'connected'}<Card.Action><Badge>Activo</Badge></Card.Action>{/if}
	</Card.Header>

	{#if data && !data.configured}
		<Card.Content><p class="text-sm text-muted-foreground">La conexión con Facebook se está terminando de configurar en el servidor. Muy pronto podrás usarla.</p></Card.Content>
	{:else if data?.status === 'choosing' && data.choices.length}
		<Card.Content>
			<p class="mb-3 font-medium">¿En qué página publicamos?</p>
			<RadioGroup bind:value={pick} class="gap-2">
				{#each data.choices as c}
					<Label class="rounded-lg border p-3 {pick === c.id ? 'border-primary' : ''}"><RadioGroupItem value={c.id} />{c.name}</Label>
				{/each}
			</RadioGroup>
		</Card.Content>
		<Card.Footer class="gap-2"><Button disabled={!pick || busy} onclick={choose}>Usar esta página</Button><Button variant="outline" disabled={busy} onclick={connect}>Volver a autorizar</Button></Card.Footer>
	{:else if data?.status === 'connected'}
		<Card.Content><p class="text-sm text-muted-foreground">Se comparten solos los artículos que se publiquen desde ahora; los anteriores los puedes compartir abajo. Si un artículo queda en borrador, se comparte hasta que lo publiques.</p></Card.Content>
		<Card.Footer class="gap-2"><Button variant="outline" disabled={busy} onclick={connect}>Cambiar de página</Button><Button variant="destructive" disabled={busy} onclick={() => (askDisconnect = true)}>Desconectar</Button></Card.Footer>
	{:else if data?.status === 'broken'}
		<Card.Content><Alert.Root variant="destructive"><Alert.Description>{data.error || 'Facebook ya no deja publicar en tu página.'} Suele pasar si cambiaste tu contraseña de Facebook o quitaste el permiso.</Alert.Description></Alert.Root></Card.Content>
		<Card.Footer class="gap-2"><Button disabled={busy} onclick={connect}>Volver a conectar</Button><Button variant="destructive" disabled={busy} onclick={() => (askDisconnect = true)}>Desconectar</Button></Card.Footer>
	{:else if data}
		<Card.Content>
			<ul class="list-disc pl-5 text-sm text-muted-foreground">
				<li>Inicias sesión con Facebook y eliges la página de tu negocio.</li>
				<li>Cuando un artículo se publica, Faro lo comparte con un texto corto, la portada y el enlace.</li>
				<li>Solo publica en la página que elijas; nunca en tu perfil personal.</li>
			</ul>
		</Card.Content>
		<Card.Footer><Button disabled={busy} onclick={connect}>Conectar Facebook</Button></Card.Footer>
	{/if}
</Card.Root>

{#if data?.status === 'connected' && data.unshared.length}
	<Card.Root>
		<Card.Header>
			<Card.Title>Artículos que aún no están en Facebook</Card.Title>
			<Card.Description>Los que publicaste antes de conectar, o los que no se pudieron compartir.</Card.Description>
			{#if data.unshared.length > 1}<Card.Action><Button size="sm" disabled={busy} onclick={() => share(data?.unshared.map((a) => a.id) || [])}>Compartir todos ({data.unshared.length})</Button></Card.Action>{/if}
		</Card.Header>
		<Card.Content>
			<Item.Group class="gap-2">
				{#each data.unshared as a}
					<Item.Root variant="outline">
						<Item.Content><Item.Title>{a.title}</Item.Title><Item.Description>Publicado el {when(a.published_at)}</Item.Description></Item.Content>
						<Item.Actions><Button variant="outline" size="sm" disabled={busy} onclick={() => share([a.id])}>Compartir</Button></Item.Actions>
					</Item.Root>
				{/each}
			</Item.Group>
			{#if data.unshared.length > 1}<p class="mt-3 text-xs text-muted-foreground">«Compartir todos» los publica uno cada 2 horas.</p>{/if}
		</Card.Content>
	</Card.Root>
{/if}

{#if data?.posts.length}
	<Card.Root>
		<Card.Header><Card.Title>Lo que se ha compartido</Card.Title></Card.Header>
		<Card.Content>
			<Item.Group class="gap-2">
				{#each data.posts as p}
					<Item.Root variant="outline">
						<Item.Content><Item.Title>{p.title}</Item.Title><Item.Description>{p.status === 'pending' && new Date(p.at) > new Date() ? 'Programado para el ' + when(p.at) : when(p.at)}{#if p.error} · {p.error}{/if}</Item.Description></Item.Content>
						<Item.Actions>
							<Badge variant={p.status === 'sent' ? 'default' : p.status === 'failed' || p.status === 'unknown' ? 'destructive' : 'secondary'}>{statusName[p.status] || p.status}</Badge>
							{#if p.url}<Button variant="outline" size="sm" href={p.url} target="_blank" rel="noopener noreferrer">Ver en Facebook</Button>{/if}
						</Item.Actions>
					</Item.Root>
				{/each}
			</Item.Group>
		</Card.Content>
	</Card.Root>
{/if}

<Card.Root class="opacity-75">
	<Card.Header>
		<Card.Title class="flex items-center gap-2"><Share2 class="size-5" />Instagram, LinkedIn y Google Business</Card.Title>
		<Card.Description>Próximamente, sin costo extra</Card.Description>
	</Card.Header>
</Card.Root>
