import { test, expect } from '@playwright/test';
const panel = 'http://localhost:18080';

test('quitar un sitio pide escribir el dominio y deja la cuenta lista para agregar otro', async ({ page }) => {
	const email = `del-${Date.now()}@faro.test`, domain = `del-${Date.now()}.faro.test`;
	await page.goto(`${panel}/dashboard`);
	await page.getByLabel('Correo').fill(email);
	await page.getByLabel('Contraseña').fill('ClaveLocalSegura2026!');
	await page.getByRole('button', { name: 'Crear cuenta', exact: true }).last().click();
	await expect(page.getByRole('heading', { name: 'Pega la dirección de tu sitio' })).toBeVisible();
	const made = await page.evaluate(async (domain) => (await fetch('/api/sites', { method: 'POST', headers: { 'content-type': 'application/json', 'x-csrf-token': localStorage.getItem('faro_csrf') || '' }, body: JSON.stringify({ name: 'Sitio a quitar', domain, canonical_url: 'http://localhost:18082', blog_prefix: '/blog', brand_description: 'x', tone: 'x' }) })).status, domain);
	expect(made).toBe(200);
	await page.goto(`${panel}/dashboard/ajustes`);
	const quitar = page.getByRole('button', { name: 'Quitar este sitio' });
	await expect(quitar).toBeDisabled(); // no se puede sin escribir el dominio
	await page.screenshot({ path: '../screenshots/panel-ajustes-peligro.png', fullPage: true });
	await page.getByLabel(/Escribe/).fill('otro.com');
	await expect(quitar).toBeDisabled();
	await page.getByLabel(/Escribe/).fill(domain);
	await expect(quitar).toBeEnabled();
	await quitar.click();
	await expect(page.getByRole('heading', { name: 'Pega la dirección de tu sitio' })).toBeVisible();
	const left = await page.evaluate(async () => (await (await fetch('/api/sites')).json()).length);
	expect(left).toBe(0);
});
