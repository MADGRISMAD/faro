<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import * as Table from '$lib/components/ui/table/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { onMount } from 'svelte';
	import { api } from '#lib/api';
	import { siteId } from '#lib/app';

	type Totals = { clicks: number; impressions: number; ctr: number; position: number };
	type Data = {
		configured: boolean; connected: boolean; property?: string; error?: string; range?: string; start?: string; end?: string;
		history?: { days_with_data: number; first?: string; last?: string } | null;
		totals?: Totals; previous?: Totals; series?: { date: string; clicks: number; impressions: number }[];
		queries?: { query: string; clicks: number; impressions: number; ctr: number; position: number }[];
		pages?: { page: string; clicks: number; impressions: number; ctr: number; position: number }[];
	};
	let range = '28d', data: Data | null = null, loading = true, error = '', notice = '', busy = false;
	const notes: Record<string, string> = { ok: 'Google conectado.', denied: 'No diste permiso en Google, así que no se conectó nada.', noproperty: 'Conectado, pero esa cuenta de Google no tiene este sitio verificado en Search Console.', error: 'No pude completar la conexión con Google. Inténtalo de nuevo.' };

	async function load() {
		loading = true;
		error = '';
		try {
			data = await api(`/sites/${$siteId}/analytics?range=${range}`);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}
	onMount(() => {
		const g = new URLSearchParams(location.search).get('google');
		if (g) notice = notes[g] || '';
		load();
	});
	async function connect() {
		busy = true;
		error = '';
		try {
			const r = await api(`/sites/${$siteId}/google/start`, { method: 'POST' });
			location.href = r.url;
		} catch (e) {
			error = (e as Error).message;
			busy = false;
		}
	}
	async function disconnect() {
		busy = true;
		try {
			await api(`/sites/${$siteId}/google`, { method: 'DELETE' });
			await load();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	const n = (v: number) => Math.round(v).toLocaleString('es-MX');
	const pct = (v: number) => (v * 100).toFixed(1) + ' %';
	// Cambio contra el periodo anterior; en posición, bajar es mejorar.
	function delta(cur: number, prev: number, lowerIsBetter = false, count = true) {
		// «nuevo» solo tiene sentido para conteos; en CTR y posición, sin periodo anterior no hay con qué comparar.
		if (!prev) return cur ? { text: count ? 'nuevo' : 'sin periodo anterior', good: true } : { text: '—', good: true };
		const d = ((cur - prev) / prev) * 100;
		return { text: (d >= 0 ? '+' : '') + d.toFixed(0) + ' %', good: lowerIsBetter ? d <= 0 : d >= 0 };
	}
	function spark(values: number[]) {
		if (values.length < 2) return '';
		const max = Math.max(...values, 1);
		return values.map((v, i) => `${(i / (values.length - 1)) * 100},${30 - (v / max) * 28}`).join(' ');
	}
	let t: Totals | undefined, p: Totals | undefined;
	// En una propiedad de dominio, Google junta varios hosts (con y sin www): si hay más de uno, se muestra el host para no repetir «/».
	$: multiHost = new Set((data?.pages || []).map((x) => x.page.replace(/^https?:\/\//, '').split('/')[0])).size > 1;
	const pageLabel = (u: string, withHost: boolean) => {
		const m = u.match(/^https?:\/\/([^/]+)(\/.*)?$/);
		return m ? (withHost ? m[1] : '') + (m[2] || '/') : u;
	};
	$: t = data?.totals;
	$: p = data?.previous;
	$: cards = t && p ? [
		['Impresiones', n(t.impressions), delta(t.impressions, p.impressions), 'Veces que Google te mostró', (data?.series || []).map((x) => x.impressions)],
		['Clics', n(t.clicks), delta(t.clicks, p.clicks), 'Visitas que llegaron desde Google', (data?.series || []).map((x) => x.clicks)],
		['CTR', pct(t.ctr), delta(t.ctr, p.ctr, false, false), 'Qué tan seguido hacen clic', []],
		['Posición media', t.position ? '#' + t.position.toFixed(1) : '—', delta(t.position, p.position, true, false), 'Tu lugar promedio en Google', []]
	] as [string, string, { text: string; good: boolean }, string, number[]][] : [];
</script>

<div class="flex flex-wrap items-start justify-between gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Analytics</h1>
		<p class="text-sm text-muted-foreground">Datos reales de Google Search Console{#if data?.start} · del {data.start} al {data.end}{/if}</p>
	</div>
	{#if data?.connected && data.property}
		<div class="flex gap-2">{#each ['7d', '28d', '90d'] as r}<Button size="sm" variant={range === r ? 'default' : 'secondary'} onclick={() => { range = r; load(); }}>{r}</Button>{/each}</div>
	{/if}
</div>
{#if notice}<Alert.Root role="status"><Alert.Description>{notice}</Alert.Description></Alert.Root>{/if}
{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}

{#if loading}
	<Alert.Root role="status"><Spinner /><Alert.Description>Pidiendo tus datos a Google…</Alert.Description></Alert.Root>
{:else if data && !data.connected}
	<Card.Root class="max-w-2xl">
		<Card.Header>
			<Card.Title class="text-xl">Conecta Google Search Console</Card.Title>
			<Card.Description>Así ves cuántas veces Google te muestra, cuántos clics recibes y en qué lugar sales, con los datos reales de Google. Solo lectura: Faro no puede cambiar nada de tu cuenta. Lee la <a class="underline" href="https://www.hyperionmkt.com/faro/privacidad" target="_blank" rel="noopener">política de privacidad</a>.</Card.Description>
		</Card.Header>
		<Card.Content class="flex flex-col gap-4">
			<ul class="list-disc pl-5 text-sm text-muted-foreground"><li>Tu sitio debe estar verificado en <b>search.google.com/search-console</b> con esa cuenta.</li><li>Google tarda unos 2-3 días en reportar, y un sitio nuevo necesita semanas para aparecer.</li></ul>
			{#if !data.configured}<Alert.Root variant="destructive" role="status"><Alert.Description>Esta función todavía no está configurada en este servidor: faltan las credenciales de Google (ver <code>docs/DATOS.md</code>).</Alert.Description></Alert.Root>{/if}
		</Card.Content>
		{#if data.configured}<Card.Footer><Button onclick={connect} disabled={busy}>{busy ? 'Abriendo Google…' : 'Conectar con Google'}</Button></Card.Footer>{/if}
	</Card.Root>
{:else if data && !data.property}
	<Card.Root class="max-w-2xl">
		<Card.Content><Alert.Root variant="destructive" role="alert"><Alert.Description>{data.error}</Alert.Description></Alert.Root></Card.Content>
		<Card.Footer class="gap-2"><Button onclick={connect} disabled={busy}>Conectar otra cuenta</Button><Button variant="outline" onclick={disconnect} disabled={busy}>Desconectar</Button></Card.Footer>
	</Card.Root>
{:else if data}
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		{#each cards as [label, value, d, hint, series]}
			<Card.Root>
				<Card.Header>
					<Card.Description>{label}</Card.Description>
					<Card.Title class="text-3xl">{value}</Card.Title>
					<span class="text-xs {d.good ? '' : 'text-destructive'}">{d.text} <span class="text-muted-foreground">vs periodo anterior</span></span>
				</Card.Header>
				<Card.Content>
					{#if series.length > 1}<svg viewBox="0 0 100 32" preserveAspectRatio="none" class="h-9 w-full text-primary" aria-hidden="true"><polyline points={spark(series)} fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" vector-effect="non-scaling-stroke" /></svg>{:else}<span class="text-xs text-muted-foreground">{hint}</span>{/if}
				</Card.Content>
			</Card.Root>
		{/each}
	</div>
	{#if !(data.totals?.impressions)}
		<Alert.Root>
			<Alert.Title>La conexión funciona, pero Google no reporta impresiones en este periodo.</Alert.Title>
			<Alert.Description>
				{#if data.history && data.history.days_with_data > 0}
					Google sí tiene datos de este sitio: {data.history.days_with_data} días entre el {data.history.first} y el {data.history.last}. Prueba con otro periodo (90d).
				{:else}
					En los últimos 16 meses Google no tiene ni una impresión registrada para <b>{data.property}</b>. Es lo normal si la propiedad se verificó hace poco, si el sitio es nuevo o si todavía casi no sale en búsquedas: Search Console empieza a llenarse uno o dos días después de verificar y un sitio nuevo tarda semanas en aparecer. Faro consulta a Google en vivo cada hora, así que los números aparecerán solos.
				{/if}
			</Alert.Description>
		</Alert.Root>
	{/if}
	<Card.Root>
		<Card.Header><Card.Title>Palabras con las que te encuentran</Card.Title></Card.Header>
		<Card.Content>
			<Table.Root>
				<Table.Header><Table.Row><Table.Head>Búsqueda</Table.Head><Table.Head class="text-right">Clics</Table.Head><Table.Head class="text-right">Impresiones</Table.Head><Table.Head class="text-right">CTR</Table.Head><Table.Head class="text-right">Posición</Table.Head></Table.Row></Table.Header>
				<Table.Body>
					{#each data.queries || [] as q}<Table.Row><Table.Cell>{q.query}</Table.Cell><Table.Cell class="text-right tabular-nums">{n(q.clicks)}</Table.Cell><Table.Cell class="text-right tabular-nums">{n(q.impressions)}</Table.Cell><Table.Cell class="text-right tabular-nums">{pct(q.ctr)}</Table.Cell><Table.Cell class="text-right tabular-nums">#{q.position.toFixed(1)}</Table.Cell></Table.Row>{:else}<Table.Row><Table.Cell colspan={5} class="text-muted-foreground">Sin datos todavía.</Table.Cell></Table.Row>{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
	<Card.Root>
		<Card.Header><Card.Title>Páginas más vistas en Google</Card.Title></Card.Header>
		<Card.Content>
			<Table.Root>
				<Table.Header><Table.Row><Table.Head>Página</Table.Head><Table.Head class="text-right">Clics</Table.Head><Table.Head class="text-right">Impresiones</Table.Head><Table.Head class="text-right">CTR</Table.Head><Table.Head class="text-right">Posición</Table.Head></Table.Row></Table.Header>
				<Table.Body>
					{#each data.pages || [] as q}<Table.Row><Table.Cell>{pageLabel(q.page, multiHost)}</Table.Cell><Table.Cell class="text-right tabular-nums">{n(q.clicks)}</Table.Cell><Table.Cell class="text-right tabular-nums">{n(q.impressions)}</Table.Cell><Table.Cell class="text-right tabular-nums">{pct(q.ctr)}</Table.Cell><Table.Cell class="text-right tabular-nums">#{q.position.toFixed(1)}</Table.Cell></Table.Row>{:else}<Table.Row><Table.Cell colspan={5} class="text-muted-foreground">Sin datos todavía.</Table.Cell></Table.Row>{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
	<p class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">Propiedad: {data.property} · <Button variant="outline" size="sm" onclick={disconnect} disabled={busy}>Desconectar Google</Button></p>
{/if}
