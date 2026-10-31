<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '#lib/api';
	import { boot } from '#lib/app';
	import * as Card from '$lib/components/ui/card/index.js';
	import * as Alert from '$lib/components/ui/alert/index.js';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { Progress } from '$lib/components/ui/progress/index.js';
	import { Spinner } from '$lib/components/ui/spinner/index.js';
	import { ArrowLeft, Eye, EyeOff } from '@lucide/svelte';

	export let initial: 'login' | 'register' = 'register';
	let mode = initial;
	let email = '', password = '', error = '', busy = false, ok = false, show = false, touched = false, caps = false;

	onMount(() => (mode = initial));
	$: validEmail = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email.trim());
	$: missing = Math.max(0, 12 - password.length);
	$: strength = Math.min(100, (password.length / 12) * 100); // 12 caracteres llenan la barra
	$: emailBad = touched && !validEmail && !ok;
	$: passBad = touched && missing > 0 && !ok;
	$: title = ok ? '¡Listo!' : mode === 'login' ? 'Qué bueno verte de nuevo' : 'Crea tu cuenta';
	$: sub = ok ? 'Entrando a tu panel…' : mode === 'login' ? 'Entra para ver tu calendario y tus artículos.' : 'Pega tu sitio y en minutos tienes tu primer calendario.';

	function switchTo(m: 'login' | 'register') {
		mode = m;
		error = '';
		touched = false;
	}
	async function submit() {
		touched = true;
		error = '';
		if (!validEmail || missing > 0) {
			error = !validEmail ? 'Revisa tu correo: parece que le falta algo.' : `La contraseña necesita 12 caracteres o más (te faltan ${missing}).`;
			return;
		}
		busy = true;
		try {
			const r = await api(`/auth/${mode}`, { method: 'POST', body: JSON.stringify({ email: email.trim(), password }) });
			localStorage.setItem('faro_csrf', r.csrf);
			ok = true;
			await new Promise((res) => setTimeout(res, 900));
			await boot();
			password = '';
		} catch (e) {
			error = (e as Error).message;
		} finally {
			busy = false;
		}
	}
	const keys = (e: KeyboardEvent) => (caps = e.getModifierState?.('CapsLock') ?? false);
</script>

<main class="flex min-h-svh flex-col items-center justify-center gap-4 p-4">
	<div class="flex w-full max-w-sm flex-col gap-4">
		<Button variant="ghost" size="sm" href="/" class="w-fit"><ArrowLeft />Volver al inicio</Button>
		<Card.Root>
			<Card.Header>
				<Badge variant="secondary" class="w-fit">Faro by Hyperion</Badge>
				<Card.Title class="text-2xl"><h1>{title}</h1></Card.Title>
				<Card.Description>{sub}</Card.Description>
			</Card.Header>
			<Card.Content class="flex flex-col gap-4">
				<Tabs.Root value={mode} onValueChange={(v) => switchTo(v as 'login' | 'register')}>
					<Tabs.List class="w-full" aria-label="Acceso">
						<Tabs.Trigger value="register" disabled={busy || ok}>Crear cuenta</Tabs.Trigger>
						<Tabs.Trigger value="login" disabled={busy || ok}>Entrar</Tabs.Trigger>
					</Tabs.List>
				</Tabs.Root>
				<form class="flex flex-col gap-4" onsubmit={(e) => { e.preventDefault(); submit(); }} novalidate>
					<div class="grid gap-2">
						<Label for="email">Correo</Label>
						<Input id="email" type="email" bind:value={email} onblur={() => email && (touched = true)} autocomplete="email" placeholder="tu@correo.com" aria-invalid={emailBad} />
					</div>
					<div class="grid gap-2">
						<Label for="password">Contraseña</Label>
						<div class="relative">
							<Input id="password" type={show ? 'text' : 'password'} bind:value={password} onkeydown={keys} onkeyup={keys} autocomplete={mode === 'login' ? 'current-password' : 'new-password'} placeholder="Mínimo 12 caracteres" aria-invalid={passBad} class="pr-10" />
							<Button type="button" variant="ghost" size="icon-sm" class="absolute top-1/2 right-1 -translate-y-1/2" aria-label={show ? 'Ocultar clave' : 'Mostrar clave'} aria-pressed={show} onclick={() => (show = !show)}>
								{#if show}<EyeOff />{:else}<Eye />{/if}
							</Button>
						</div>
					</div>
					{#if mode === 'register' && !ok}
						<div class="grid gap-2" aria-live="polite">
							<Progress value={strength} max={100} />
							<p class="text-xs text-muted-foreground">{missing === 0 ? 'Contraseña lista' : `Te faltan ${missing} caracteres`}</p>
						</div>
					{/if}
					{#if caps}<p class="text-sm text-muted-foreground" role="status">Tienes el bloqueo de mayúsculas activado.</p>{/if}
					{#if error}<Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>{/if}
					<Button type="submit" disabled={busy || ok}>
						{#if busy || ok}<Spinner />{/if}
						{ok ? 'Listo' : busy ? (mode === 'register' ? 'Creando tu cuenta…' : 'Entrando…') : mode === 'register' ? 'Crear cuenta' : 'Entrar'}
					</Button>
				</form>
			</Card.Content>
			<Card.Footer class="text-sm text-muted-foreground">
				{mode === 'register' ? '¿Ya tienes cuenta?' : '¿Aún no tienes cuenta?'}
				<Button variant="link" size="sm" disabled={busy || ok} onclick={() => switchTo(mode === 'register' ? 'login' : 'register')}>{mode === 'register' ? 'Inicia sesión' : 'Crea una'}</Button>
			</Card.Footer>
		</Card.Root>
	</div>
</main>
