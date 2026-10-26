import { test, expect } from '@playwright/test';
import { execSync } from 'node:child_process';
const panel = 'http://localhost:18080';

// La lectura por URL no puede abrir localhost (SSRF a propósito), así que el estado "la IA ya analizó, falta confirmar" se siembra en la base local.
const sql = (q: string) => execSync('docker-compose exec -T postgres psql -U faro -d faro -v ON_ERROR_STOP=1', { cwd: '..', input: q });
const topics = ['Cómo elegir un taller', 'Cuándo cambiar frenos', 'Precio de una afinación', 'Aceite sintético o mineral', 'Revisión antes de viaje'].map((title) => ({ title, keyword: title.toLowerCase(), intent: 'informational', why: 'Responde una duda frecuente.' }));

test('asistente de Brand DNA: la IA sugiere, el cliente ajusta y solo al confirmar empieza a escribir', async ({ page }) => {
	const email = `wiz-${Date.now()}@faro.test`, domain = `wiz-${Date.now()}.faro.test`;
	await page.goto(`${panel}/dashboard`);
	await page.getByLabel('Correo').fill(email);
	await page.getByLabel('Contraseña').fill('ClaveLocalSegura2026!');
	await page.getByRole('button', { name: 'Crear cuenta', exact: true }).last().click();
	await expect(page.getByRole('heading', { name: 'Pega la dirección de tu sitio' })).toBeVisible();
	const made = await page.evaluate(async (domain) => (await fetch('/api/sites', { method: 'POST', headers: { 'content-type': 'application/json', 'x-csrf-token': localStorage.getItem('faro_csrf') || '' }, body: JSON.stringify({ name: 'Taller Wizard', domain, canonical_url: 'http://localhost:18082', blog_prefix: '/blog', brand_description: 'x', tone: 'x' }) })).status, domain);
	expect(made).toBe(200);
	const profile = JSON.stringify({ name: 'Taller Wizard', business: 'Taller mecánico familiar en Tijuana.', audience: 'Conductores de la ciudad.', tone: 'Cercano y claro', location: 'Tijuana', offerings: ['Frenos', 'Afinación'], topics, sensitive: false, sensitive_reason: '', platform: 'other' }).replace(/'/g, "''");
	sql(`UPDATE sites SET brand_confirmed_at=NULL, autopilot=true, research='${profile}'::jsonb WHERE domain='${domain}';
	INSERT INTO jobs(tenant_id,site_id,idempotency_key,topic,business,audience,kind,status) SELECT tenant_id,id,'research-'||id,'x','','','research','done' FROM sites WHERE domain='${domain}';`);

	await page.goto(`${panel}/dashboard`);
	await expect(page.getByRole('heading', { name: 'Tu negocio' })).toBeVisible();
	await expect(page.locator('.side')).toHaveCount(0); // en la primera vez no hay panel: solo el asistente
	await expect(page.getByText('Sugerido por la IA')).toBeVisible();
	await page.screenshot({ path: '../screenshots/wizard-1.png', fullPage: true });
	await page.locator('textarea').first().fill('Taller mecánico familiar con 20 años en Tijuana.');
	await page.getByRole('button', { name: 'Se ve bien, continuar' }).click();
	await expect(page.getByRole('heading', { name: 'Tu público' })).toBeVisible();
	await page.getByRole('button', { name: 'Se ve bien, continuar' }).click();
	await expect(page.getByRole('heading', { name: 'Tu voz' })).toBeVisible();
	await page.getByRole('button', { name: 'Se ve bien, continuar' }).click();
	await expect(page.getByRole('heading', { name: 'Imágenes' })).toBeVisible();
	await page.getByRole('button', { name: 'Se ve bien, continuar' }).click();
	await expect(page.getByRole('heading', { name: 'Temas' })).toBeVisible();
	await page.screenshot({ path: '../screenshots/wizard-5.png', fullPage: true });
	await page.getByRole('checkbox').first().uncheck(); // quita «Cómo elegir un taller»
	await page.getByRole('button', { name: 'Se ve bien, continuar' }).click();
	await expect(page.getByRole('heading', { name: 'Ritmo' })).toBeVisible();
	await page.screenshot({ path: '../screenshots/wizard-6.png', fullPage: true });

	// antes de confirmar no se escribió nada
	const before = await page.evaluate(async () => { const s = await (await fetch('/api/sites')).json(); const c = await (await fetch(`/api/sites/${s[0].id}/calendar`)).json(); return c.articles.length + c.generating.length; });
	expect(before).toBe(0);
	await page.getByRole('button', { name: 'Empezar a escribir' }).click();
	await expect(page.locator('.side')).toBeVisible();

	const saved = await page.evaluate(async () => { const s = await (await fetch('/api/sites')).json(); return (await fetch(`/api/sites/${s[0].id}/research`)).json(); });
	expect(saved.brand_confirmed).toBe(true);
	expect(saved.research.business).toBe('Taller mecánico familiar con 20 años en Tijuana.'); // lo que editó el cliente
	expect(saved.research.topics).toHaveLength(4); // el tema que quitó ya no está
	await expect(page.getByText('Escribiendo ahora').or(page.locator('.post').first()).first()).toBeVisible({ timeout: 30000 });
	// la página de inicio ya con la cola, el calendario y el primer artículo escrito por la IA
	await page.setViewportSize({ width: 1440, height: 900 });
	await expect(page.getByRole('link', { name: 'Ver borrador' })).toBeVisible({ timeout: 40000 }); // el texto simulado es corto: queda en borrador, como debe
	await page.evaluate(async () => { // se aprueba y publica para ver también la tarjeta del último artículo
		const h = { 'content-type': 'application/json', 'x-csrf-token': localStorage.getItem('faro_csrf') || '' };
		const [a] = await (await fetch('/api/articles')).json();
		const ok = (await fetch(`/api/articles/${a.id}/approve`, { method: 'POST', headers: h })).status;
		const pub = (await fetch(`/api/articles/${a.id}/publish`, { method: 'POST', headers: h })).status;
		return [ok, pub];
	}).then((st) => expect(st).toEqual([200, 200]));
	await page.reload();
	await page.waitForTimeout(2500);
	await page.screenshot({ path: '/tmp/wiz-debug.png', fullPage: true });
	await expect(page.locator('.latest')).toBeVisible({ timeout: 20000 });
	await expect(page.locator('.qitem').first()).toBeVisible();
	await expect(page.locator('.mc .day.on').first()).toBeVisible();
	await page.screenshot({ path: '../screenshots/panel-inicio-completo.png', fullPage: true });
});
