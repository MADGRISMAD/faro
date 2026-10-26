<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '#lib/api';
	import { boot } from '#lib/app';

	export let initial: 'login' | 'register' = 'register';
	let mode = initial;
	let email = '', password = '', error = '', busy = false, ok = false, show = false, touched = false, shake = 0, caps = false;

	onMount(() => (mode = initial));
	$: validEmail = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(email.trim());
	$: missing = Math.max(0, 12 - password.length);
	$: strength = Math.min(4, Math.floor(password.length / 3)); // 12 caracteres llenan las 4 barras
	$: emailBad = touched && !validEmail && !ok;
	$: passBad = touched && missing > 0 && !ok;
	$: face = ok ? '/mascota/saluda.png' : error ? '/mascota/asombro.png' : '/mascota/animada.gif';
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
			shake++;
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
			shake++;
		} finally {
			busy = false;
		}
	}
	const keys = (e: KeyboardEvent) => (caps = e.getModifierState?.('CapsLock') ?? false);
</script>

<main class="auth">
	<a class="back" href="/"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M19 12H5m6-6-6 6 6 6" /></svg>Volver al inicio</a>
	<div class="shell-card" class:a={shake % 2 === 1} class:b={shake > 0 && shake % 2 === 0} class:done={ok}>
		<div class="face">
			{#key face}<img src={face} width="96" height="96" alt="" />{/key}
		</div>
		<p class="eyebrow">Faro <span class="by">by Hyperion</span></p>
		{#key title}<h1>{title}</h1>{/key}
		<p class="sub">{sub}</p>

		<div class="seg" role="group" aria-label="Acceso" data-mode={mode}>
			<span class="thumb-pill"></span>
			<button type="button" class:on={mode === 'register'} disabled={busy || ok} on:click={() => switchTo('register')}>Crear cuenta</button>
			<button type="button" class:on={mode === 'login'} disabled={busy || ok} on:click={() => switchTo('login')}>Entrar</button>
		</div>

		<form on:submit|preventDefault={submit} novalidate>
			<label class:bad={emailBad} class:good={touched && validEmail}>Correo
				<span class="in"><svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="5" width="18" height="14" rx="3" /><path d="m4 8 8 5 8-5" /></svg>
					<input type="email" bind:value={email} on:blur={() => email && (touched = true)} autocomplete="email" placeholder="tu@correo.com" aria-invalid={emailBad} />
					{#if validEmail}<i class="tick" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="m5 12.5 4.5 4.5L19 7.5" /></svg></i>{/if}
				</span>
			</label>
			<label class:bad={passBad} class:good={touched && missing === 0}>Contraseña
				<span class="in"><svg viewBox="0 0 24 24" aria-hidden="true"><rect x="5" y="11" width="14" height="9" rx="2.5" /><path d="M8 11V8a4 4 0 0 1 8 0v3" /></svg>
					<input type={show ? 'text' : 'password'} bind:value={password} on:keydown={keys} on:keyup={keys} autocomplete={mode === 'login' ? 'current-password' : 'new-password'} placeholder="Mínimo 12 caracteres" aria-invalid={passBad} />
					<button type="button" class="eye" aria-label={show ? 'Ocultar clave' : 'Mostrar clave'} aria-pressed={show} on:click={() => (show = !show)}>
						<svg viewBox="0 0 24 24" aria-hidden="true">{#if show}<path d="M3 3l18 18M10.6 6.2A9.6 9.6 0 0 1 12 6c5 0 8.5 4 9.5 6a12 12 0 0 1-2.6 3.3M6.5 7.8A12.5 12.5 0 0 0 2.5 12C3.5 14 7 18 12 18c1.2 0 2.3-.2 3.3-.6M9.9 9.9a3 3 0 0 0 4.2 4.2" />{:else}<path d="M2.5 12C3.5 10 7 6 12 6s8.5 4 9.5 6c-1 2-4.5 6-9.5 6S3.5 14 2.5 12Z" /><circle cx="12" cy="12" r="3" />{/if}</svg>
					</button>
				</span>
			</label>
			{#if mode === 'register' && !ok}
				<div class="meter" aria-live="polite">
					<span class="bars">{#each [1, 2, 3, 4] as n}<i class:on={strength >= n}></i>{/each}</span>
					<small class:full={missing === 0}>{missing === 0 ? 'Contraseña lista' : `Te faltan ${missing} caracteres`}</small>
				</div>
			{/if}
			{#if caps}<p class="hint" role="status">Tienes el bloqueo de mayúsculas activado.</p>{/if}
			{#if error}<p class="msg" role="alert"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9" /><path d="M12 7.5V13M12 16.5h.01" /></svg>{error}</p>{/if}
			<button class="go" class:ok disabled={busy || ok}>
				{#if ok}<svg class="check" viewBox="0 0 24 24" aria-hidden="true"><path d="m5 12.5 4.5 4.5L19 7.5" /></svg>{:else if busy}<span class="spinner"></span>{/if}
				<span>{ok ? 'Listo' : busy ? (mode === 'register' ? 'Creando tu cuenta…' : 'Entrando…') : mode === 'register' ? 'Crear cuenta' : 'Entrar'}</span>
			</button>
		</form>
		<p class="swap">{mode === 'register' ? '¿Ya tienes cuenta?' : '¿Aún no tienes cuenta?'} <button type="button" disabled={busy || ok} on:click={() => switchTo(mode === 'register' ? 'login' : 'register')}>{mode === 'register' ? 'Inicia sesión' : 'Crea una'}</button></p>
	</div>
</main>

<style>
	.auth{max-width:none;min-height:100vh;display:grid;place-content:center;justify-items:center;gap:20px;padding:28px 18px 60px;background:#fcfafb}
	.back{justify-self:start;display:inline-flex;gap:8px;align-items:center;font-weight:800;font-size:.9rem;color:var(--mut);text-decoration:none;position:absolute;top:24px;left:max(18px,calc((100vw - 1160px)/2))}
	.back svg,.in svg,.msg svg,.go svg{width:18px;height:18px;fill:none;stroke:currentColor;stroke-width:2;stroke-linecap:round;stroke-linejoin:round}
	.back:hover{color:var(--accent)}
	.shell-card{width:min(440px,100%);padding:30px 34px 28px;border-radius:44px;background:var(--surface);box-shadow:var(--out);text-align:center;position:relative}
	.shell-card.a{animation:shakeA .45s}.shell-card.b{animation:shakeB .45s}
	@keyframes shakeA{20%,60%{transform:translateX(-9px)}40%,80%{transform:translateX(9px)}}
	@keyframes shakeB{20%,60%{transform:translateX(-9px)}40%,80%{transform:translateX(9px)}}
	.face{height:96px;margin-top:-76px;display:grid;place-items:center}
	.face img{height:96px;width:auto;filter:drop-shadow(0 10px 10px rgba(194,24,91,.22));animation:pop .35s cubic-bezier(.3,1.6,.5,1)}
	.done .face img{animation:hop .7s ease-in-out infinite}
	@keyframes pop{from{transform:scale(.7);opacity:0}}
	@keyframes hop{50%{transform:translateY(-10px)}}
	.eyebrow{margin:12px 0 0}
	.by{text-transform:none;letter-spacing:.02em;font-size:.82em;font-weight:800;color:var(--mut)}
	h1{margin:6px 0 0;font-size:1.9rem;animation:rise .3s ease}
	@keyframes rise{from{opacity:0;transform:translateY(6px)}}
	.sub{margin:8px 0 22px;color:var(--mut);font-size:.98rem;line-height:1.5}
	.seg{position:relative;display:grid;grid-template-columns:1fr 1fr;padding:5px;border-radius:99px;box-shadow:var(--in);margin-bottom:22px}
	.thumb-pill{position:absolute;top:5px;bottom:5px;left:5px;width:calc(50% - 5px);border-radius:99px;background:var(--surface);box-shadow:var(--out-s);transition:transform .25s cubic-bezier(.4,1.3,.6,1)}
	.seg[data-mode=login] .thumb-pill{transform:translateX(100%)}
	.seg button{position:relative;border:0;background:none;padding:11px 0;border-radius:99px;font-weight:800;color:var(--mut);transition:color .2s}
	.seg button.on{color:var(--accent)}
	form{display:grid;gap:16px;text-align:left}
	label{gap:8px}
	.in{display:flex;align-items:center;gap:10px;padding:0 14px;border-radius:99px;box-shadow:var(--in-s);color:var(--mut);border:2px solid transparent;transition:border-color .2s,box-shadow .2s}
	.in:focus-within{border-color:var(--accent-l);color:var(--accent)}
	.in input{box-shadow:none;background:transparent;padding:14px 0;border-radius:0;color:var(--ink);font-weight:600}
	.in input:focus-visible{outline:none}
	label.bad .in{border-color:var(--accent);color:var(--accent);animation:nudge .3s}
	label.good .in{color:#15803d}
	@keyframes nudge{30%{transform:translateX(-4px)}60%{transform:translateX(4px)}}
	.tick{display:grid;place-items:center;width:22px;height:22px;border-radius:50%;background:#d6f5e3;color:#15803d;flex:none;animation:pop .25s}
	.tick svg{width:13px;height:13px;stroke-width:3.2}
	.eye{border:0;background:none;color:inherit;padding:4px;border-radius:50%;display:grid;place-items:center}
	.eye:hover{color:var(--accent)}
	.meter{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-top:-4px}
	.bars{display:flex;gap:6px;flex:1}.bars i{height:7px;flex:1;border-radius:99px;box-shadow:var(--in-s);transition:background .25s}
	.bars i.on{background:var(--accent-l);box-shadow:none}
	.meter small{color:var(--mut);font-weight:700;font-size:.8rem;white-space:nowrap}.meter small.full{color:#15803d}
	.bars:has(i:nth-child(4).on) i.on{background:#34a96a}
	.hint{margin:0;font-size:.85rem;font-weight:700;color:var(--accent-d)}
	.msg{display:flex;gap:10px;align-items:flex-start;margin:0;padding:13px 16px;border-radius:22px;background:var(--accent-xl);color:#8a0b3e;font-weight:700;font-size:.92rem;line-height:1.4;box-shadow:var(--in-s);animation:rise .25s}
	.msg svg{flex:none;margin-top:1px}
	.go{display:flex;align-items:center;justify-content:center;gap:10px;height:54px;border:0;border-radius:99px;background:var(--accent);color:#fff;font-weight:800;font-size:1.05rem;box-shadow:var(--puff);transition:background .25s,transform .12s}
	.go:hover:not(:disabled){transform:translateY(-1px)}
	.go:disabled{cursor:wait;opacity:1}
	.go.ok{background:#15803d;box-shadow:6px 8px 18px rgba(21,128,61,.3),-5px -5px 12px var(--light)}
	.check{stroke-width:3;animation:pop .3s}
	.spinner{width:18px;height:18px;border:3px solid rgba(255,255,255,.4);border-top-color:#fff;border-radius:50%;animation:sp .7s linear infinite}
	.swap{margin:20px 0 0;color:var(--mut);font-size:.92rem}
	.swap button{border:0;background:none;padding:0;color:var(--accent);font-weight:800;text-decoration:underline}
	@media(max-width:480px){.shell-card{padding:28px 20px 24px}.back{position:static;justify-self:start}.auth{place-content:start;padding-top:20px}.face{margin-top:-66px}}
	@media(prefers-reduced-motion:reduce){.shell-card.a,.shell-card.b,.face img,.done .face img,h1,.msg,.tick,.spinner,.check,label.bad .in{animation:none}.thumb-pill,.seg button,.go{transition:none}.spinner{border-top-color:rgba(255,255,255,.4)}}
</style>
