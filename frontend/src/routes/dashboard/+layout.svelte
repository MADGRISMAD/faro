<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import Wizard from '../../lib/Wizard.svelte';
	import Auth from '../../lib/Auth.svelte';
	import { api } from '#lib/api';
	import { user, ready, sites, siteId, addingSite, research, boot, loadSites, selectSite, stopPolling, refreshSite } from '#lib/app';
	import * as Sidebar from '$lib/components/ui/sidebar/index.js';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { Avatar, AvatarFallback } from '$lib/components/ui/avatar/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import { NativeSelect, NativeSelectOption } from '$lib/components/ui/native-select/index.js';
	import { House, Dna, CalendarDays, FileText, Plug, Share2, ChartColumn, Search, Settings, Plus, LogOut } from '@lucide/svelte';

	let initialMode: 'login' | 'register' = 'register';
	let url = '', error = '', busy = false;

	onMount(() => {
		here();
		if (new URLSearchParams(location.search).get('modo') === 'entrar') initialMode = 'login'; // viene del botón «Iniciar sesión» de la portada
		boot();
		// al volver a la pestaña se ve al momento lo que el autopiloto hizo mientras tanto (artículos y portadas)
		document.addEventListener('visibilitychange', onVisible);
	});
	onDestroy(() => {
		stopPolling();
		if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', onVisible);
	});

	const onVisible = () => document.visibilityState === 'visible' && $user && refreshSite();
	async function analyze() {
		busy = true;
		error = '';
		try {
			const r = await api('/sites/analyze', { method: 'POST', body: JSON.stringify({ url }) });
			url = '';
			siteId.set(r.site_id);
			addingSite.set(false);
			await loadSites();
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	async function logout() {
		try {
			await api('/auth/logout', { method: 'POST' });
		} catch {}
		localStorage.removeItem('faro_csrf');
		user.set(null);
		sites.set([]);
		siteId.set('');
	}
	const nav = [
		{ href: '/dashboard', label: 'Inicio', icon: House },
		{ href: '/dashboard/brand', label: 'Brand DNA', icon: Dna },
		{ href: '/dashboard/calendario', label: 'Calendario', icon: CalendarDays },
		{ href: '/dashboard/articulos', label: 'Artículos', icon: FileText },
		{ href: '/dashboard/conectar', label: 'Conectar mi sitio', icon: Plug },
		{ href: '/dashboard/redes', label: 'Redes sociales', icon: Share2 }
	];
	const data = [
		{ href: '/dashboard/analytics', label: 'Analytics', icon: ChartColumn },
		{ href: '/dashboard/palabras-clave', label: 'Palabras clave', icon: Search }
	];
	let path = '';
	const here = () => (path = location.pathname.replace(/\/$/, ''));
	$: active = $sites.find((s) => s.id === $siteId);
	// Un sitio que la IA acaba de analizar espera a que el cliente revise su Brand DNA antes de escribir nada.
	$: settingUp = !!$research && !$research.brand_confirmed;
	$: reading = settingUp && ($research?.status === 'queued' || $research?.status === 'running');
</script>

<svelte:head><title>Faro by Hyperion</title><meta name="robots" content="noindex" /></svelte:head>
<svelte:window onpopstate={here} />

{#if !$ready}
	<main class="flex min-h-svh items-center justify-center gap-2 text-muted-foreground"><Spinner />Cargando…</main>
{:else if !$user}
	<Auth initial={initialMode} />
{:else if !$sites.length || $addingSite}
	<main class="mx-auto flex min-h-svh w-full max-w-xl items-center p-4">
		<Card.Root class="w-full">
			<Card.Header>
				<Badge variant="secondary" class="w-fit">Empieza aquí</Badge>
				<Card.Title class="text-2xl"><h1>Pega la dirección de tu sitio</h1></Card.Title>
				<Card.Description>La IA lee tu sitio, entiende tu negocio, elige los temas y empieza a escribir. No tienes que llenar nada más.</Card.Description>
			</Card.Header>
			<Card.Content class="flex flex-col gap-3">
				<form onsubmit={(e) => { e.preventDefault(); (analyze)(); }} class="flex flex-wrap gap-2">
					<Input bind:value={url} required maxlength={300} placeholder="tusitio.com" aria-label="Dirección de tu sitio" autocapitalize="off" autocorrect="off" spellcheck={false} class="min-w-52 flex-1" />
					<Button type="submit" disabled={busy}>{busy ? 'Enviando…' : 'Analizar mi sitio'}</Button>
				</form>
				<p class="text-sm text-muted-foreground">Sirve con o sin https, con www o sin él.</p>
				{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}
			</Card.Content>
			{#if $addingSite && $sites.length}
				<Card.Footer><Button variant="secondary" onclick={() => addingSite.set(false)}>Cancelar</Button></Card.Footer>
			{/if}
		</Card.Root>
	</main>
{:else if reading}
	<main class="mx-auto flex min-h-svh w-full max-w-xl items-center p-4">
		<Card.Root class="w-full">
			<Card.Header>
				<Badge variant="secondary" class="w-fit">{active?.domain}</Badge>
				<Card.Title class="text-2xl"><h1>Leyendo tu sitio…</h1></Card.Title>
			</Card.Header>
			<Card.Content>
				<Alert.Root role="status"><Spinner /><Alert.Description>La IA está entendiendo tu negocio y eligiendo los temas. Tarda menos de un minuto; después podrás revisar todo antes de que escriba nada.</Alert.Description></Alert.Root>
			</Card.Content>
		</Card.Root>
	</main>
{:else if settingUp && $research?.status === 'failed'}
	<main class="mx-auto flex min-h-svh w-full max-w-xl items-center p-4">
		<Card.Root class="w-full">
			<Card.Header><Card.Title class="text-2xl"><h1>No pude leer tu sitio</h1></Card.Title></Card.Header>
			<Card.Content><Alert.Root variant="destructive" role="alert"><Alert.Description>{$research?.error}</Alert.Description></Alert.Root></Card.Content>
			<Card.Footer><Button onclick={() => addingSite.set(true)}>Probar con otra dirección</Button></Card.Footer>
		</Card.Root>
	</main>
{:else if settingUp && $research?.status === 'done'}
	<Wizard />
{:else}
	<Sidebar.Provider>
		<Sidebar.Root>
			<Sidebar.Header>
				<a href="/dashboard" class="flex flex-col px-2 pt-1 leading-tight"><span class="text-lg font-semibold">Faro</span><span class="text-xs text-muted-foreground">by Hyperion</span></a>
				<NativeSelect aria-label="Sitio" value={$siteId} onchange={(e) => selectSite(e.currentTarget.value)} class="w-full">
					{#each $sites as s}<NativeSelectOption value={s.id}>{s.name}</NativeSelectOption>{/each}
				</NativeSelect>
				<p class="px-2 text-xs text-muted-foreground">{active?.domain}</p>
			</Sidebar.Header>
			<Sidebar.Content>
				<Sidebar.Group>
					<Sidebar.GroupContent>
						<Sidebar.Menu>
							{#each nav as item}
								<Sidebar.MenuItem>
									<Sidebar.MenuButton isActive={path === item.href}>
										{#snippet child({ props })}
											<a {...props} href={item.href} onclick={() => (path = item.href)}>
												<item.icon />
												<span>{item.label}</span>
												{#if item.href === '/dashboard/conectar' && $research && !$research.connected}<Badge class="ml-auto">Falta</Badge>{/if}
											</a>
										{/snippet}
									</Sidebar.MenuButton>
								</Sidebar.MenuItem>
							{/each}
						</Sidebar.Menu>
					</Sidebar.GroupContent>
				</Sidebar.Group>
				<Sidebar.Group>
					<Sidebar.GroupLabel>Datos</Sidebar.GroupLabel>
					<Sidebar.GroupContent>
						<Sidebar.Menu>
							{#each data as item}
								<Sidebar.MenuItem>
									<Sidebar.MenuButton isActive={path === item.href}>
										{#snippet child({ props })}
											<a {...props} href={item.href} onclick={() => (path = item.href)}><item.icon /><span>{item.label}</span></a>
										{/snippet}
									</Sidebar.MenuButton>
								</Sidebar.MenuItem>
							{/each}
						</Sidebar.Menu>
					</Sidebar.GroupContent>
				</Sidebar.Group>
			</Sidebar.Content>
			<Sidebar.Footer>
				<Sidebar.Menu>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton isActive={path === '/dashboard/ajustes'}>
							{#snippet child({ props })}
								<a {...props} href="/dashboard/ajustes" onclick={() => (path = '/dashboard/ajustes')}><Settings /><span>Ajustes</span></a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
					<Sidebar.MenuItem>
						<Sidebar.MenuButton>
							{#snippet child({ props })}
								<a {...props} href="#nuevo" onclick={(e) => { e.preventDefault(); (() => addingSite.set(true))(); }}><Plus /><span>Añadir otro sitio</span></a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				</Sidebar.Menu>
				<Separator />
				<div class="flex items-center gap-2 px-2 text-sm">
					<Avatar class="size-8"><AvatarFallback>{($user.email || '?').slice(0, 2).toUpperCase()}</AvatarFallback></Avatar>
					<span class="truncate">{$user.email}</span>
				</div>
				<Button variant="secondary" size="sm" onclick={logout}><LogOut />Cerrar sesión</Button>
				<p class="text-center text-xs text-muted-foreground"><a class="underline" href="https://www.hyperionmkt.com/faro/privacidad" target="_blank" rel="noopener">Privacidad</a> · <a class="underline" href="https://www.hyperionmkt.com/faro/terminos" target="_blank" rel="noopener">Condiciones</a></p>
			</Sidebar.Footer>
			<Sidebar.Rail />
		</Sidebar.Root>
		<Sidebar.Inset>
			<header class="flex h-12 items-center gap-2 border-b px-4 md:hidden"><Sidebar.Trigger /><span class="font-semibold">Faro</span></header>
			<div class="mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 md:p-8"><slot /></div>
		</Sidebar.Inset>
	</Sidebar.Provider>
{/if}
