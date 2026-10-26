<script lang="ts">
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

<div class="view wide">
	<h1>Calendario de contenido</h1>
	<p class="sub">La IA planea y escribe tus artículos; aquí ves cuándo sale cada uno.</p>
	<div class="panel">
		<div class="row" style="margin-bottom:16px">
			<h2 style="margin:0;font-size:1.4rem">{monthName}</h2>
			<div style="display:flex;gap:8px;align-items:center">
				<button class="btn ghost small" aria-label="Mes anterior" on:click={() => shift(-1)}>‹</button>
				<button class="btn ghost small" on:click={() => { year = now.getFullYear(); month = now.getMonth(); }}>Hoy</button>
				<button class="btn ghost small" aria-label="Mes siguiente" on:click={() => shift(1)}>›</button>
			</div>
		</div>
		<div class="cal">
			{#each ['Lun', 'Mar', 'Mié', 'Jue', 'Vie', 'Sáb', 'Dom'] as d}<div class="dow">{d}</div>{/each}
			{#each days as d}
				{@const items = byDay[key(d)] || []}
				<div class="cell" class:off={d.getMonth() !== month} class:today={key(d) === today}>
					<div class="n"><span>{d.getDate()}</span>{#if key(d) === today}<b style="color:#c45530">Hoy</b>{/if}</div>
					{#each items as it}
						{#if it.kind === 'article'}
							<a class="mini" href={`/dashboard/articulos?abrir=${it.id}`} style="text-decoration:none;color:inherit">
								{#if it.has_image}<img class="thumb" src={`/api/articles/${it.id}/image`} alt="" loading="lazy" />{/if}
								<b>{it.title}</b><span class="muted">{label[it.status || ''] || ''}</span>
							</a>
						{:else if it.kind === 'generating'}
							<div class="mini"><b>{it.title}</b><span class="muted"><span class="spin"></span>Escribiendo…</span></div>
						{:else}
							<div class="mini plan"><b>{it.title}</b><span>Planeado · {it.note}</span></div>
						{/if}
					{/each}
				</div>
			{/each}
		</div>
		<div class="legend"><span><span class="chip published">Publicado</span></span><span><span class="chip draft">Borrador</span></span><span><span class="spin"></span>Escribiendo</span><span><span class="chip plan">Planeado por la IA</span></span></div>
	</div>
</div>
