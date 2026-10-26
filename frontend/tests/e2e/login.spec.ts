import { test, expect } from '@playwright/test';
const panel = 'http://localhost:18080';

test('la portada tiene «Iniciar sesión» que lleva al acceso y, con sesión abierta, «Ir a mi panel»', async ({ page }) => {
	const email = `login-${Date.now()}@faro.test`, pass = 'ClaveLocalSegura2026!';
	await page.goto(`${panel}/dashboard`);
	await page.getByLabel('Correo').fill(email);
	await page.getByLabel('Contraseña').fill(pass);
	await page.getByRole('button', { name: 'Crear cuenta', exact: true }).last().click();
	await expect(page.getByRole('heading', { name: 'Pega la dirección de tu sitio' })).toBeVisible();
	// se cierra la sesión para probar el acceso desde cero
	await page.evaluate(async () => { await fetch('/api/auth/logout', { method: 'POST', headers: { 'content-type': 'application/json', 'x-csrf-token': localStorage.getItem('faro_csrf') || '' } }); localStorage.removeItem('faro_csrf'); });

	await page.goto(`${panel}/`);
	await expect(page.getByRole('heading', { name: 'Tu blog, escrito solo.' })).toBeVisible();
	await expect(page.locator('header.top')).toBeVisible();
	await page.screenshot({ path: '../screenshots/portada.png', fullPage: true });
	await page.locator('header.top').getByRole('link', { name: 'Iniciar sesión' }).click();
	await expect(page).toHaveURL(/\/dashboard\?modo=entrar/);
	await page.getByLabel('Correo').fill(email);
	await page.getByLabel('Contraseña').fill(pass);
	await page.getByRole('button', { name: 'Entrar', exact: true }).last().click();
	await expect(page.getByRole('heading', { name: 'Pega la dirección de tu sitio' })).toBeVisible();

	await page.goto(`${panel}/`);
	await expect(page.getByRole('link', { name: 'Ir a mi panel' }).first()).toBeVisible();
	await expect(page.getByRole('link', { name: 'Iniciar sesión' })).toHaveCount(0);

	// una página de blog de un cliente no lleva ninguna marca de Faro arriba
	const html = await (await page.request.get(`${panel}/blog`, { headers: { 'x-faro-domain': 'no-existe.test' } })).text();
	expect(html).not.toContain('FaroContenido.');
});
