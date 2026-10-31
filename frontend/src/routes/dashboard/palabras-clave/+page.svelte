<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Table from '$lib/components/ui/table/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { onMount } from 'svelte';
	import { api } from '#lib/api';
	import { siteId, intentName } from '#lib/app';

	type Metrics = { volume: number | null; competition: string | null; competition_index: number | null; cpc: number | null; difficulty: number | null };
	type Row = { title: string; keyword: string; intent: string; why: string; written: boolean; metrics?: Metrics };
	let rows: Row[] = [], configured = false, loading = true, busy = false, error = '', notice = '';

	async function load() {
		try {
			const d = await api(`/sites/${$siteId}/keywords`);
			rows = d.keywords;
			configured = d.configured;
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}
	onMount(load);
	async function refresh() {
		busy = true;
		error = notice = '';
		try {
			const r = await api(`/sites/${$siteId}/keywords/refresh`, { method: 'POST' });
			notice = r.updated ? `Se actualizaron ${r.updated} palabras.` : 'Los datos ya estaban al día (se renuevan cada 30 días).';
			await load();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	const level = (d: number) => (d < 30 ? 'easy' : d < 60 ? 'medium' : 'hard');
	const levelName: Record<string, string> = { easy: 'fácil', medium: 'media', hard: 'difícil' };
	$: withData = rows.filter((r) => r.metrics?.volume != null).length;
</script>

<div class="flex flex-wrap items-start justify-between gap-3">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Palabras clave</h1>
		<p class="text-sm text-muted-foreground">Los temas que la IA eligió, ordenados por oportunidad real: mucho volumen y poca dificultad.</p>
	</div>
	{#if configured}<Button variant="outline" onclick={refresh} disabled={busy}>{busy ? 'Actualizando…' : 'Actualizar datos'}</Button>{/if}
</div>
{#if notice}<Alert.Root role="status"><Alert.Description>{notice}</Alert.Description></Alert.Root>{/if}
{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}
{#if !loading && !configured}
	<Alert.Root>
		<Alert.Title>Sin datos de búsqueda todavía.</Alert.Title>
		<Alert.Description>Para ver volumen mensual, dificultad y costo por clic reales hay que conectar un proveedor de palabras clave (DataForSEO) en el servidor; mientras tanto no se muestra ningún número para no inventarlos. Pasos en <code>docs/DATOS.md</code>.</Alert.Description>
	</Alert.Root>
{:else if !loading && withData === 0 && rows.length}
	<Alert.Root><Alert.Description>Aún no hay datos para estas palabras. Pulsa «Actualizar datos».</Alert.Description></Alert.Root>
{/if}
<Card.Root>
	<Card.Content>
		{#if loading}<span class="flex items-center gap-2 text-muted-foreground"><Spinner />Cargando…</span>
		{:else}
			<Table.Root>
				<Table.Header>
					<Table.Row><Table.Head>Tema</Table.Head><Table.Head class="text-right">Búsquedas / mes</Table.Head><Table.Head class="text-right">Dificultad</Table.Head><Table.Head class="text-right">Costo por clic</Table.Head><Table.Head>Estado</Table.Head></Table.Row>
				</Table.Header>
				<Table.Body>
					{#each rows as r}
						<Table.Row>
							<Table.Cell><b>{r.title}</b><br /><span class="text-xs text-muted-foreground">{r.keyword} · {intentName[r.intent] || r.intent}</span></Table.Cell>
							<Table.Cell class="text-right tabular-nums">{r.metrics?.volume != null ? r.metrics.volume.toLocaleString('es-MX') : '—'}</Table.Cell>
							<Table.Cell class="text-right">{#if r.metrics?.difficulty != null}<Badge variant={level(r.metrics.difficulty) === 'easy' ? 'secondary' : level(r.metrics.difficulty) === 'medium' ? 'outline' : 'destructive'}>{r.metrics.difficulty}</Badge> <span class="text-xs text-muted-foreground">{levelName[level(r.metrics.difficulty)]}</span>{:else}—{/if}</Table.Cell>
							<Table.Cell class="text-right tabular-nums">{r.metrics?.cpc != null ? 'US$' + r.metrics.cpc.toFixed(2) : '—'}</Table.Cell>
							<Table.Cell>{r.written ? 'Escrito o en cola' : 'Pendiente'}</Table.Cell>
						</Table.Row>
					{:else}
						<Table.Row><Table.Cell colspan={5} class="text-muted-foreground">Aún no hay temas: la IA los elige al analizar tu sitio.</Table.Cell></Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
			{#if withData}<p class="mt-4 text-xs text-muted-foreground">Fuente: Google Ads vía DataForSEO, México y español. La dificultad va de 0 a 100 (probabilidad de entrar al top 10).</p>{/if}
		{/if}
	</Card.Content>
</Card.Root>
