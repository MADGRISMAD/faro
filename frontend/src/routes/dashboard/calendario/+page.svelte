<script lang="ts">
	import * as Card from '$lib/components/ui/card/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { ChevronLeft, ChevronRight } from '@lucide/svelte';
	import { calendar, intentName } from '#lib/app';

	const now = new Date();
	let year = now.getFullYear(), month = now.getMonth();
	const key = (d: Date) => `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`;
	$: monthName = new Date(year, month, 1).toLocaleDateString('es-MX', { month: 'long', year: 'numeric' }).replace(/^./, (c) => c.toUpperCase()).replace(' de ', ' ');
	$: offset = (new Date(year, month, 1).getDay() + 6) % 7; // la semana empieza en lunes
	$: weeks = Math.ceil((offset + new Date(year, month + 1, 0).getDate()) / 7);
	$: days = Array.from({ length: weeks * 7 }, (_, i) => new Date(year, month, 1 - offset + i));
	$: byDay = (() => {
		const m: Record<string, { kind: string; title: string; status?: string; id?: string; has_image?: boolean; note?: string }[]> = {};
		const put = (at: string, v: any) => (m[key(new Date(at))] ||= []).push(v);
		$calendar?.articles.forEach((a) => put(a.at, { kind: 'article', title: a.title, status: a.status, id: a.id, has_image: a.has_image }));
		$calendar?.generating.forEach((g) => put(g.at, { kind: 'generating', title: g.title }));
		$calendar?.upcoming.forEach((u) => put(u.at, { kind: 'plan', title: u.title, note: intentName[u.intent] }));
		return m;
	})();
	const shift = (n: number) => {
		const d = new Date(year, month + n, 1);
		year = d.getFullYear();
		month = d.getMonth();
	};
	const today = key(now);
	const label: Record<string, string> = { published: 'Publicado', draft: 'Borrador', approved: 'Aprobado' };
</script>

<div>
	<h1 class="text-2xl font-semibold tracking-tight">Calendario de contenido</h1>
	<p class="text-sm text-muted-foreground">La IA planea y escribe tus artículos; aquí ves cuándo sale cada uno.</p>
</div>
<Card.Root>
	<Card.Header>
		<Card.Title>{monthName}</Card.Title>
		<Card.Action class="flex items-center gap-2">
			<Button variant="outline" size="icon-sm" aria-label="Mes anterior" onclick={() => shift(-1)}><ChevronLeft /></Button>
			<Button variant="outline" size="sm" onclick={() => { year = now.getFullYear(); month = now.getMonth(); }}>Hoy</Button>
			<Button variant="outline" size="icon-sm" aria-label="Mes siguiente" onclick={() => shift(1)}><ChevronRight /></Button>
		</Card.Action>
	</Card.Header>
	<Card.Content>
		<div class="grid grid-cols-7 gap-1 sm:gap-2">
			{#each ['Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb', 'Dom'] as d}<div class="pb-1 text-center text-xs text-muted-foreground">{d}</div>{/each}
			{#each days as d}
				{@const items = byDay[key(d)] || []}
				<div class="flex min-h-20 flex-col gap-1 rounded-lg border p-1 sm:min-h-32 sm:p-2 {d.getMonth() !== month ? 'opacity-50' : ''} {key(d) === today ? 'ring-2 ring-ring' : ''}">
					<div class="flex justify-between text-xs text-muted-foreground"><span>{d.getDate()}</span>{#if key(d) === today}<b class="text-foreground">Hoy</b>{/if}</div>
					{#each items as it}
						{#if it.kind === 'article'}
							<a class="flex flex-col gap-1 rounded-md border bg-card p-1.5 text-xs hover:bg-muted" href={`/dashboard/articulos?abrir=${it.id}`}>
								{#if it.has_image}<img class="hidden aspect-video w-full rounded object-cover sm:block" src={`/api/articles/${it.id}/image`} alt="" loading="lazy" />{/if}
								<b class="line-clamp-2">{it.title}</b><span class="text-muted-foreground">{label[it.status || ''] || ''}</span>
							</a>
						{:else if it.kind === 'generating'}
							<div class="flex flex-col gap-1 rounded-md border bg-card p-1.5 text-xs"><b class="line-clamp-2">{it.title}</b><span class="flex items-center gap-1 text-muted-foreground"><Spinner class="size-3" />Escribiendo…</span></div>
						{:else}
							<div class="flex flex-col gap-1 rounded-md border border-dashed p-1.5 text-xs text-muted-foreground"><b class="line-clamp-2 text-foreground">{it.title}</b><span>Planeado · {it.note}</span></div>
						{/if}
					{/each}
				</div>
			{/each}
		</div>
	</Card.Content>
	<Card.Footer class="flex flex-wrap gap-3 text-sm text-muted-foreground">
		<Badge>Publicado</Badge><Badge variant="secondary">Borrador</Badge><span class="flex items-center gap-1"><Spinner class="size-3" />Escribiendo</span><Badge variant="outline">Planeado por la IA</Badge>
	</Card.Footer>
</Card.Root>
